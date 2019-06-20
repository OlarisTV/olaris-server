// Package googlecloudstorage provides an interface to Google Cloud Storage
package googlecloudstorage

/*
Notes

Can't set Updated but can set Metadata on object creation

Patch needs full_control not just read_write

FIXME Patch/Delete/Get isn't working with files with spaces in - giving 404 error
- https://code.google.com/p/google-api-go-client/issues/detail?id=64
*/

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ncw/rclone/fs"
	"github.com/ncw/rclone/fs/config"
	"github.com/ncw/rclone/fs/config/configmap"
	"github.com/ncw/rclone/fs/config/configstruct"
	"github.com/ncw/rclone/fs/config/obscure"
	"github.com/ncw/rclone/fs/fserrors"
	"github.com/ncw/rclone/fs/fshttp"
	"github.com/ncw/rclone/fs/hash"
	"github.com/ncw/rclone/fs/walk"
	"github.com/ncw/rclone/lib/oauthutil"
	"github.com/ncw/rclone/lib/pacer"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"

	// NOTE: This API is deprecated
	storage "google.golang.org/api/storage/v1"
)

const (
	rcloneClientID              = "202264815644.apps.googleusercontent.com"
	rcloneEncryptedClientSecret = "Uj7C9jGfb9gmeaV70Lh058cNkWvepr-Es9sBm0zdgil7JaOWF1VySw"
	timeFormatIn                = time.RFC3339
	timeFormatOut               = "2006-01-02T15:04:05.000000000Z07:00"
	metaMtime                   = "mtime" // key to store mtime under in metadata
	listChunks                  = 1000    // chunk size to read directory listings
	minSleep                    = 10 * time.Millisecond
)

var (
	// Description of how to auth for this app
	storageConfig = &oauth2.Config{
		Scopes:       []string{storage.DevstorageFullControlScope},
		Endpoint:     google.Endpoint,
		ClientID:     rcloneClientID,
		ClientSecret: obscure.MustReveal(rcloneEncryptedClientSecret),
		RedirectURL:  oauthutil.TitleBarRedirectURL,
	}
)

// Register with Fs
func init() {
	fs.Register(&fs.RegInfo{
		Name:        "google cloud storage",
		Prefix:      "gcs",
		Description: "Google Cloud Storage (this is not Google Drive)",
		NewFs:       NewFs,
		Config: func(name string, m configmap.Mapper) {
			saFile, _ := m.Get("service_account_file")
			saCreds, _ := m.Get("service_account_credentials")
			if saFile != "" || saCreds != "" {
				return
			}
			err := oauthutil.Config("google cloud storage", name, m, storageConfig)
			if err != nil {
				log.Fatalf("Failed to configure token: %v", err)
			}
		},
		Options: []fs.Option{{
			Name: config.ConfigClientID,
			Help: "Google Application Client Id\nLeave blank normally.",
		}, {
			Name: config.ConfigClientSecret,
			Help: "Google Application Client Secret\nLeave blank normally.",
		}, {
			Name: "project_number",
			Help: "Project number.\nOptional - needed only for list/create/delete buckets - see your developer console.",
		}, {
			Name: "service_account_file",
			Help: "Service Account Credentials JSON file path\nLeave blank normally.\nNeeded only if you want use SA instead of interactive login.",
		}, {
			Name: "service_account_credentials",
			Help: "Service Account Credentials JSON blob\nLeave blank normally.\nNeeded only if you want use SA instead of interactive login.",
			Hide: fs.OptionHideBoth,
		}, {
			Name: "object_acl",
			Help: "Access Control List for new objects.",
			Examples: []fs.OptionExample{{
				Value: "authenticatedRead",
				Help:  "Object owner gets OWNER access, and all Authenticated Users get READER access.",
			}, {
				Value: "bucketOwnerFullControl",
				Help:  "Object owner gets OWNER access, and project team owners get OWNER access.",
			}, {
				Value: "bucketOwnerRead",
				Help:  "Object owner gets OWNER access, and project team owners get READER access.",
			}, {
				Value: "private",
				Help:  "Object owner gets OWNER access [default if left blank].",
			}, {
				Value: "projectPrivate",
				Help:  "Object owner gets OWNER access, and project team members get access according to their roles.",
			}, {
				Value: "publicRead",
				Help:  "Object owner gets OWNER access, and all Users get READER access.",
			}},
		}, {
			Name: "bucket_acl",
			Help: "Access Control List for new buckets.",
			Examples: []fs.OptionExample{{
				Value: "authenticatedRead",
				Help:  "Project team owners get OWNER access, and all Authenticated Users get READER access.",
			}, {
				Value: "private",
				Help:  "Project team owners get OWNER access [default if left blank].",
			}, {
				Value: "projectPrivate",
				Help:  "Project team members get access according to their roles.",
			}, {
				Value: "publicRead",
				Help:  "Project team owners get OWNER access, and all Users get READER access.",
			}, {
				Value: "publicReadWrite",
				Help:  "Project team owners get OWNER access, and all Users get WRITER access.",
			}},
		}, {
			Name: "bucket_policy_only",
			Help: `Access checks should use bucket-level IAM policies.

If you want to upload objects to a bucket with Bucket Policy Only set
then you will need to set this.

When it is set, rclone:

- ignores ACLs set on buckets
- ignores ACLs set on objects
- creates buckets with Bucket Policy Only set

Docs: https://cloud.google.com/storage/docs/bucket-policy-only
`,
			Default: false,
		}, {
			Name: "location",
			Help: "Location for the newly created buckets.",
			Examples: []fs.OptionExample{{
				Value: "",
				Help:  "Empty for default location (US).",
			}, {
				Value: "asia",
				Help:  "Multi-regional location for Asia.",
			}, {
				Value: "eu",
				Help:  "Multi-regional location for Europe.",
			}, {
				Value: "us",
				Help:  "Multi-regional location for United States.",
			}, {
				Value: "asia-east1",
				Help:  "Taiwan.",
			}, {
				Value: "asia-east2",
				Help:  "Hong Kong.",
			}, {
				Value: "asia-northeast1",
				Help:  "Tokyo.",
			}, {
				Value: "asia-south1",
				Help:  "Mumbai.",
			}, {
				Value: "asia-southeast1",
				Help:  "Singapore.",
			}, {
				Value: "australia-southeast1",
				Help:  "Sydney.",
			}, {
				Value: "europe-north1",
				Help:  "Finland.",
			}, {
				Value: "europe-west1",
				Help:  "Belgium.",
			}, {
				Value: "europe-west2",
				Help:  "London.",
			}, {
				Value: "europe-west3",
				Help:  "Frankfurt.",
			}, {
				Value: "europe-west4",
				Help:  "Netherlands.",
			}, {
				Value: "us-central1",
				Help:  "Iowa.",
			}, {
				Value: "us-east1",
				Help:  "South Carolina.",
			}, {
				Value: "us-east4",
				Help:  "Northern Virginia.",
			}, {
				Value: "us-west1",
				Help:  "Oregon.",
			}, {
				Value: "us-west2",
				Help:  "California.",
			}},
		}, {
			Name: "storage_class",
			Help: "The storage class to use when storing objects in Google Cloud Storage.",
			Examples: []fs.OptionExample{{
				Value: "",
				Help:  "Default",
			}, {
				Value: "MULTI_REGIONAL",
				Help:  "Multi-regional storage class",
			}, {
				Value: "REGIONAL",
				Help:  "Regional storage class",
			}, {
				Value: "NEARLINE",
				Help:  "Nearline storage class",
			}, {
				Value: "COLDLINE",
				Help:  "Coldline storage class",
			}, {
				Value: "DURABLE_REDUCED_AVAILABILITY",
				Help:  "Durable reduced availability storage class",
			}},
		}},
	})
}

// Options defines the configuration for this backend
type Options struct {
	ProjectNumber             string `config:"project_number"`
	ServiceAccountFile        string `config:"service_account_file"`
	ServiceAccountCredentials string `config:"service_account_credentials"`
	ObjectACL                 string `config:"object_acl"`
	BucketACL                 string `config:"bucket_acl"`
	BucketPolicyOnly          bool   `config:"bucket_policy_only"`
	Location                  string `config:"location"`
	StorageClass              string `config:"storage_class"`
}

// Fs represents a remote storage server
type Fs struct {
	name       string           // name of this remote
	root       string           // the path we are working on if any
	opt        Options          // parsed options
	features   *fs.Features     // optional features
	svc        *storage.Service // the connection to the storage server
	client     *http.Client     // authorized client
	bucket     string           // the bucket we are working on
	bucketOKMu sync.Mutex       // mutex to protect bucket OK
	bucketOK   bool             // true if we have created the bucket
	pacer      *fs.Pacer        // To pace the API calls
}

// Object describes a storage object
//
// Will definitely have info but maybe not meta
type Object struct {
	fs       *Fs       // what this object is part of
	remote   string    // The remote path
	url      string    // download path
	md5sum   string    // The MD5Sum of the object
	bytes    int64     // Bytes in the object
	modTime  time.Time // Modified time of the object
	mimeType string
}

// ------------------------------------------------------------

// Name of the remote (as passed into NewFs)
func (f *Fs) Name() string {
	return f.name
}

// Root of the remote (as passed into NewFs)
func (f *Fs) Root() string {
	if f.root == "" {
		return f.bucket
	}
	return f.bucket + "/" + f.root
}

// String converts this Fs to a string
func (f *Fs) String() string {
	if f.root == "" {
		return fmt.Sprintf("Storage bucket %s", f.bucket)
	}
	return fmt.Sprintf("Storage bucket %s path %s", f.bucket, f.root)
}

// Features returns the optional features of this Fs
func (f *Fs) Features() *fs.Features {
	return f.features
}

// shouldRetry determines whether a given err rates being retried
func shouldRetry(err error) (again bool, errOut error) {
	again = false
	if err != nil {
		if fserrors.ShouldRetry(err) {
			again = true
		} else {
			switch gerr := err.(type) {
			case *googleapi.Error:
				if gerr.Code >= 500 && gerr.Code < 600 {
					// All 5xx errors should be retried
					again = true
				} else if len(gerr.Errors) > 0 {
					reason := gerr.Errors[0].Reason
					if reason == "rateLimitExceeded" || reason == "userRateLimitExceeded" {
						again = true
					}
				}
			}
		}
	}
	return again, err
}

// Pattern to match a storage path
var matcher = regexp.MustCompile(`^([^/]*)(.*)$`)

// parseParse parses a storage 'url'
func parsePath(path string) (bucket, directory string, err error) {
	parts := matcher.FindStringSubmatch(path)
	if parts == nil {
		err = errors.Errorf("couldn't find bucket in storage path %q", path)
	} else {
		bucket, directory = parts[1], parts[2]
		directory = strings.Trim(directory, "/")
	}
	return
}

func getServiceAccountClient(credentialsData []byte) (*http.Client, error) {
	conf, err := google.JWTConfigFromJSON(credentialsData, storageConfig.Scopes...)
	if err != nil {
		return nil, errors.Wrap(err, "error processing credentials")
	}
	ctxWithSpecialClient := oauthutil.Context(fshttp.NewClient(fs.Config))
	return oauth2.NewClient(ctxWithSpecialClient, conf.TokenSource(ctxWithSpecialClient)), nil
}

// NewFs constructs an Fs from the path, bucket:path
func NewFs(name, root string, m configmap.Mapper) (fs.Fs, error) {
	var oAuthClient *http.Client

	// Parse config into Options struct
	opt := new(Options)
	err := configstruct.Set(m, opt)
	if err != nil {
		return nil, err
	}
	if opt.ObjectACL == "" {
		opt.ObjectACL = "private"
	}
	if opt.BucketACL == "" {
		opt.BucketACL = "private"
	}

	// try loading service account credentials from env variable, then from a file
	if opt.ServiceAccountCredentials == "" && opt.ServiceAccountFile != "" {
		loadedCreds, err := ioutil.ReadFile(os.ExpandEnv(opt.ServiceAccountFile))
		if err != nil {
			return nil, errors.Wrap(err, "error opening service account credentials file")
		}
		opt.ServiceAccountCredentials = string(loadedCreds)
	}
	if opt.ServiceAccountCredentials != "" {
		oAuthClient, err = getServiceAccountClient([]byte(opt.ServiceAccountCredentials))
		if err != nil {
			return nil, errors.Wrap(err, "failed configuring Google Cloud Storage Service Account")
		}
	} else {
		oAuthClient, _, err = oauthutil.NewClient(name, m, storageConfig)
		if err != nil {
			ctx := context.Background()
			oAuthClient, err = google.DefaultClient(ctx, storage.DevstorageFullControlScope)
			if err != nil {
				return nil, errors.Wrap(err, "failed to configure Google Cloud Storage")
			}
		}
	}

	bucket, directory, err := parsePath(root)
	if err != nil {
		return nil, err
	}

	f := &Fs{
		name:   name,
		bucket: bucket,
		root:   directory,
		opt:    *opt,
		pacer:  fs.NewPacer(pacer.NewGoogleDrive(pacer.MinSleep(minSleep))),
	}
	f.features = (&fs.Features{
		ReadMimeType:  true,
		WriteMimeType: true,
		BucketBased:   true,
	}).Fill(f)

	// Create a new authorized Drive client.
	f.client = oAuthClient
	f.svc, err = storage.New(f.client)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create Google Cloud Storage client")
	}

	if f.root != "" {
		f.root += "/"
		// Check to see if the object exists
		err = f.pacer.Call(func() (bool, error) {
			_, err = f.svc.Objects.Get(bucket, directory).Do()
			return shouldRetry(err)
		})
		if err == nil {
			f.root = path.Dir(directory)
			if f.root == "." {
				f.root = ""
			} else {
				f.root += "/"
			}
			// return an error with an fs which points to the parent
			return f, fs.ErrorIsFile
		}
	}
	return f, nil
}

// Return an Object from a path
//
// If it can't be found it returns the error fs.ErrorObjectNotFound.
func (f *Fs) newObjectWithInfo(remote string, info *storage.Object) (fs.Object, error) {
	o := &Object{
		fs:     f,
		remote: remote,
	}
	if info != nil {
		o.setMetaData(info)
	} else {
		err := o.readMetaData() // reads info and meta, returning an error
		if err != nil {
			return nil, err
		}
	}
	return o, nil
}

// NewObject finds the Object at remote.  If it can't be found
// it returns the error fs.ErrorObjectNotFound.
func (f *Fs) NewObject(ctx context.Context, remote string) (fs.Object, error) {
	return f.newObjectWithInfo(remote, nil)
}

// listFn is called from list to handle an object.
type listFn func(remote string, object *storage.Object, isDirectory bool) error

// list the objects into the function supplied
//
// dir is the starting directory, "" for root
//
// Set recurse to read sub directories
func (f *Fs) list(ctx context.Context, dir string, recurse bool, fn listFn) (err error) {
	root := f.root
	rootLength := len(root)
	if dir != "" {
		root += dir + "/"
	}
	list := f.svc.Objects.List(f.bucket).Prefix(root).MaxResults(listChunks)
	if !recurse {
		list = list.Delimiter("/")
	}
	for {
		var objects *storage.Objects
		err = f.pacer.Call(func() (bool, error) {
			objects, err = list.Do()
			return shouldRetry(err)
		})
		if err != nil {
			if gErr, ok := err.(*googleapi.Error); ok {
				if gErr.Code == http.StatusNotFound {
					err = fs.ErrorDirNotFound
				}
			}
			return err
		}
		if !recurse {
			var object storage.Object
			for _, prefix := range objects.Prefixes {
				if !strings.HasSuffix(prefix, "/") {
					continue
				}
				err = fn(prefix[rootLength:len(prefix)-1], &object, true)
				if err != nil {
					return err
				}
			}
		}
		for _, object := range objects.Items {
			if !strings.HasPrefix(object.Name, root) {
				fs.Logf(f, "Odd name received %q", object.Name)
				continue
			}
			remote := object.Name[rootLength:]
			// is this a directory marker?
			if (strings.HasSuffix(remote, "/") || remote == "") && object.Size == 0 {
				if recurse && remote != "" {
					// add a directory in if --fast-list since will have no prefixes
					err = fn(remote[:len(remote)-1], object, true)
					if err != nil {
						return err
					}
				}
				continue // skip directory marker
			}
			err = fn(remote, object, false)
			if err != nil {
				return err
			}
		}
		if objects.NextPageToken == "" {
			break
		}
		list.PageToken(objects.NextPageToken)
	}
	return nil
}

// Convert a list item into a DirEntry
func (f *Fs) itemToDirEntry(remote string, object *storage.Object, isDirectory bool) (fs.DirEntry, error) {
	if isDirectory {
		d := fs.NewDir(remote, time.Time{}).SetSize(int64(object.Size))
		return d, nil
	}
	o, err := f.newObjectWithInfo(remote, object)
	if err != nil {
		return nil, err
	}
	return o, nil
}

// mark the bucket as being OK
func (f *Fs) markBucketOK() {
	if f.bucket != "" {
		f.bucketOKMu.Lock()
		f.bucketOK = true
		f.bucketOKMu.Unlock()
	}
}

// listDir lists a single directory
func (f *Fs) listDir(ctx context.Context, dir string) (entries fs.DirEntries, err error) {
	// List the objects
	err = f.list(ctx, dir, false, func(remote string, object *storage.Object, isDirectory bool) error {
		entry, err := f.itemToDirEntry(remote, object, isDirectory)
		if err != nil {
			return err
		}
		if entry != nil {
			entries = append(entries, entry)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	// bucket must be present if listing succeeded
	f.markBucketOK()
	return entries, err
}

// listBuckets lists the buckets
func (f *Fs) listBuckets(dir string) (entries fs.DirEntries, err error) {
	if dir != "" {
		return nil, fs.ErrorListBucketRequired
	}
	if f.opt.ProjectNumber == "" {
		return nil, errors.New("can't list buckets without project number")
	}
	listBuckets := f.svc.Buckets.List(f.opt.ProjectNumber).MaxResults(listChunks)
	for {
		var buckets *storage.Buckets
		err = f.pacer.Call(func() (bool, error) {
			buckets, err = listBuckets.Do()
			return shouldRetry(err)
		})
		if err != nil {
			return nil, err
		}
		for _, bucket := range buckets.Items {
			d := fs.NewDir(bucket.Name, time.Time{})
			entries = append(entries, d)
		}
		if buckets.NextPageToken == "" {
			break
		}
		listBuckets.PageToken(buckets.NextPageToken)
	}
	return entries, nil
}

// List the objects and directories in dir into entries.  The
// entries can be returned in any order but should be for a
// complete directory.
//
// dir should be "" to list the root, and should not have
// trailing slashes.
//
// This should return ErrDirNotFound if the directory isn't
// found.
func (f *Fs) List(ctx context.Context, dir string) (entries fs.DirEntries, err error) {
	if f.bucket == "" {
		return f.listBuckets(dir)
	}
	return f.listDir(ctx, dir)
}

// ListR lists the objects and directories of the Fs starting
// from dir recursively into out.
//
// dir should be "" to start from the root, and should not
// have trailing slashes.
//
// This should return ErrDirNotFound if the directory isn't
// found.
//
// It should call callback for each tranche of entries read.
// These need not be returned in any particular order.  If
// callback returns an error then the listing will stop
// immediately.
//
// Don't implement this unless you have a more efficient way
// of listing recursively that doing a directory traversal.
func (f *Fs) ListR(ctx context.Context, dir string, callback fs.ListRCallback) (err error) {
	if f.bucket == "" {
		return fs.ErrorListBucketRequired
	}
	list := walk.NewListRHelper(callback)
	err = f.list(ctx, dir, true, func(remote string, object *storage.Object, isDirectory bool) error {
		entry, err := f.itemToDirEntry(remote, object, isDirectory)
		if err != nil {
			return err
		}
		return list.Add(entry)
	})
	if err != nil {
		return err
	}
	// bucket must be present if listing succeeded
	f.markBucketOK()
	return list.Flush()
}

// Put the object into the bucket
//
// Copy the reader in to the new object which is returned
//
// The new object may have been created if an error is returned
func (f *Fs) Put(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) (fs.Object, error) {
	// Temporary Object under construction
	o := &Object{
		fs:     f,
		remote: src.Remote(),
	}
	return o, o.Update(ctx, in, src, options...)
}

// PutStream uploads to the remote path with the modTime given of indeterminate size
func (f *Fs) PutStream(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) (fs.Object, error) {
	return f.Put(ctx, in, src, options...)
}

// Mkdir creates the bucket if it doesn't exist
func (f *Fs) Mkdir(ctx context.Context, dir string) (err error) {
	f.bucketOKMu.Lock()
	defer f.bucketOKMu.Unlock()
	if f.bucketOK {
		return nil
	}
	// List something from the bucket to see if it exists.  Doing it like this enables the use of a
	// service account that only has the "Storage Object Admin" role.  See #2193 for details.

	err = f.pacer.Call(func() (bool, error) {
		_, err = f.svc.Objects.List(f.bucket).MaxResults(1).Do()
		return shouldRetry(err)
	})
	if err == nil {
		// Bucket already exists
		f.bucketOK = true
		return nil
	} else if gErr, ok := err.(*googleapi.Error); ok {
		if gErr.Code != http.StatusNotFound {
			return errors.Wrap(err, "failed to get bucket")
		}
	} else {
		return errors.Wrap(err, "failed to get bucket")
	}

	if f.opt.ProjectNumber == "" {
		return errors.New("can't make bucket without project number")
	}

	bucket := storage.Bucket{
		Name:         f.bucket,
		Location:     f.opt.Location,
		StorageClass: f.opt.StorageClass,
	}
	if f.opt.BucketPolicyOnly {
		bucket.IamConfiguration = &storage.BucketIamConfiguration{
			BucketPolicyOnly: &storage.BucketIamConfigurationBucketPolicyOnly{
				Enabled: true,
			},
		}
	}
	err = f.pacer.Call(func() (bool, error) {
		insertBucket := f.svc.Buckets.Insert(f.opt.ProjectNumber, &bucket)
		if !f.opt.BucketPolicyOnly {
			insertBucket.PredefinedAcl(f.opt.BucketACL)
		}
		_, err = insertBucket.Do()
		return shouldRetry(err)
	})
	if err == nil {
		f.bucketOK = true
	}
	return err
}

// Rmdir deletes the bucket if the fs is at the root
//
// Returns an error if it isn't empty: Error 409: The bucket you tried
// to delete was not empty.
func (f *Fs) Rmdir(ctx context.Context, dir string) (err error) {
	f.bucketOKMu.Lock()
	defer f.bucketOKMu.Unlock()
	if f.root != "" || dir != "" {
		return nil
	}
	err = f.pacer.Call(func() (bool, error) {
		err = f.svc.Buckets.Delete(f.bucket).Do()
		return shouldRetry(err)
	})
	if err == nil {
		f.bucketOK = false
	}
	return err
}

// Precision returns the precision
func (f *Fs) Precision() time.Duration {
	return time.Nanosecond
}

// Copy src to this remote using server side copy operations.
//
// This is stored with the remote path given
//
// It returns the destination Object and a possible error
//
// Will only be called if src.Fs().Name() == f.Name()
//
// If it isn't possible then return fs.ErrorCantCopy
func (f *Fs) Copy(ctx context.Context, src fs.Object, remote string) (fs.Object, error) {
	err := f.Mkdir(ctx, "")
	if err != nil {
		return nil, err
	}
	srcObj, ok := src.(*Object)
	if !ok {
		fs.Debugf(src, "Can't copy - not same remote type")
		return nil, fs.ErrorCantCopy
	}

	// Temporary Object under construction
	dstObj := &Object{
		fs:     f,
		remote: remote,
	}

	srcBucket := srcObj.fs.bucket
	srcObject := srcObj.fs.root + srcObj.remote
	dstBucket := f.bucket
	dstObject := f.root + remote
	var newObject *storage.Object
	err = f.pacer.Call(func() (bool, error) {
		newObject, err = f.svc.Objects.Copy(srcBucket, srcObject, dstBucket, dstObject, nil).Do()
		return shouldRetry(err)
	})
	if err != nil {
		return nil, err
	}
	// Set the metadata for the new object while we have it
	dstObj.setMetaData(newObject)
	return dstObj, nil
}

// Hashes returns the supported hash sets.
func (f *Fs) Hashes() hash.Set {
	return hash.Set(hash.MD5)
}

// ------------------------------------------------------------

// Fs returns the parent Fs
func (o *Object) Fs() fs.Info {
	return o.fs
}

// Return a string version
func (o *Object) String() string {
	if o == nil {
		return "<nil>"
	}
	return o.remote
}

// Remote returns the remote path
func (o *Object) Remote() string {
	return o.remote
}

// Hash returns the Md5sum of an object returning a lowercase hex string
func (o *Object) Hash(ctx context.Context, t hash.Type) (string, error) {
	if t != hash.MD5 {
		return "", hash.ErrUnsupported
	}
	return o.md5sum, nil
}

// Size returns the size of an object in bytes
func (o *Object) Size() int64 {
	return o.bytes
}

// setMetaData sets the fs data from a storage.Object
func (o *Object) setMetaData(info *storage.Object) {
	o.url = info.MediaLink
	o.bytes = int64(info.Size)
	o.mimeType = info.ContentType

	// Read md5sum
	md5sumData, err := base64.StdEncoding.DecodeString(info.Md5Hash)
	if err != nil {
		fs.Logf(o, "Bad MD5 decode: %v", err)
	} else {
		o.md5sum = hex.EncodeToString(md5sumData)
	}

	// read mtime out of metadata if available
	mtimeString, ok := info.Metadata[metaMtime]
	if ok {
		modTime, err := time.Parse(timeFormatIn, mtimeString)
		if err == nil {
			o.modTime = modTime
			return
		}
		fs.Debugf(o, "Failed to read mtime from metadata: %s", err)
	}

	// Fallback to the Updated time
	modTime, err := time.Parse(timeFormatIn, info.Updated)
	if err != nil {
		fs.Logf(o, "Bad time decode: %v", err)
	} else {
		o.modTime = modTime
	}
}

// readMetaData gets the metadata if it hasn't already been fetched
//
// it also sets the info
func (o *Object) readMetaData() (err error) {
	if !o.modTime.IsZero() {
		return nil
	}
	var object *storage.Object
	err = o.fs.pacer.Call(func() (bool, error) {
		object, err = o.fs.svc.Objects.Get(o.fs.bucket, o.fs.root+o.remote).Do()
		return shouldRetry(err)
	})
	if err != nil {
		if gErr, ok := err.(*googleapi.Error); ok {
			if gErr.Code == http.StatusNotFound {
				return fs.ErrorObjectNotFound
			}
		}
		return err
	}
	o.setMetaData(object)
	return nil
}

// ModTime returns the modification time of the object
//
// It attempts to read the objects mtime and if that isn't present the
// LastModified returned in the http headers
func (o *Object) ModTime(ctx context.Context) time.Time {
	err := o.readMetaData()
	if err != nil {
		// fs.Logf(o, "Failed to read metadata: %v", err)
		return time.Now()
	}
	return o.modTime
}

// Returns metadata for an object
func metadataFromModTime(modTime time.Time) map[string]string {
	metadata := make(map[string]string, 1)
	metadata[metaMtime] = modTime.Format(timeFormatOut)
	return metadata
}

// SetModTime sets the modification time of the local fs object
func (o *Object) SetModTime(ctx context.Context, modTime time.Time) (err error) {
	// This only adds metadata so will perserve other metadata
	object := storage.Object{
		Bucket:   o.fs.bucket,
		Name:     o.fs.root + o.remote,
		Metadata: metadataFromModTime(modTime),
	}
	var newObject *storage.Object
	err = o.fs.pacer.Call(func() (bool, error) {
		newObject, err = o.fs.svc.Objects.Patch(o.fs.bucket, o.fs.root+o.remote, &object).Do()
		return shouldRetry(err)
	})
	if err != nil {
		return err
	}
	o.setMetaData(newObject)
	return nil
}

// Storable returns a boolean as to whether this object is storable
func (o *Object) Storable() bool {
	return true
}

// Open an object for read
func (o *Object) Open(ctx context.Context, options ...fs.OpenOption) (in io.ReadCloser, err error) {
	req, err := http.NewRequest("GET", o.url, nil)
	if err != nil {
		return nil, err
	}
	fs.OpenOptionAddHTTPHeaders(req.Header, options)
	var res *http.Response
	err = o.fs.pacer.Call(func() (bool, error) {
		res, err = o.fs.client.Do(req)
		if err == nil {
			err = googleapi.CheckResponse(res)
			if err != nil {
				_ = res.Body.Close() // ignore error
			}
		}
		return shouldRetry(err)
	})
	if err != nil {
		return nil, err
	}
	_, isRanging := req.Header["Range"]
	if !(res.StatusCode == http.StatusOK || (isRanging && res.StatusCode == http.StatusPartialContent)) {
		_ = res.Body.Close() // ignore error
		return nil, errors.Errorf("bad response: %d: %s", res.StatusCode, res.Status)
	}
	return res.Body, nil
}

// Update the object with the contents of the io.Reader, modTime and size
//
// The new object may have been created if an error is returned
func (o *Object) Update(ctx context.Context, in io.Reader, src fs.ObjectInfo, options ...fs.OpenOption) error {
	err := o.fs.Mkdir(ctx, "")
	if err != nil {
		return err
	}
	modTime := src.ModTime(ctx)

	object := storage.Object{
		Bucket:      o.fs.bucket,
		Name:        o.fs.root + o.remote,
		ContentType: fs.MimeType(ctx, src),
		Metadata:    metadataFromModTime(modTime),
	}
	var newObject *storage.Object
	err = o.fs.pacer.CallNoRetry(func() (bool, error) {
		insertObject := o.fs.svc.Objects.Insert(o.fs.bucket, &object).Media(in, googleapi.ContentType("")).Name(object.Name)
		if !o.fs.opt.BucketPolicyOnly {
			insertObject.PredefinedAcl(o.fs.opt.ObjectACL)
		}
		newObject, err = insertObject.Do()
		return shouldRetry(err)
	})
	if err != nil {
		return err
	}
	// Set the metadata for the new object while we have it
	o.setMetaData(newObject)
	return nil
}

// Remove an object
func (o *Object) Remove(ctx context.Context) (err error) {
	err = o.fs.pacer.Call(func() (bool, error) {
		err = o.fs.svc.Objects.Delete(o.fs.bucket, o.fs.root+o.remote).Do()
		return shouldRetry(err)
	})
	return err
}

// MimeType of an Object if known, "" otherwise
func (o *Object) MimeType(ctx context.Context) string {
	return o.mimeType
}

// Check the interfaces are satisfied
var (
	_ fs.Fs          = &Fs{}
	_ fs.Copier      = &Fs{}
	_ fs.PutStreamer = &Fs{}
	_ fs.ListRer     = &Fs{}
	_ fs.Object      = &Object{}
	_ fs.MimeTyper   = &Object{}
)
