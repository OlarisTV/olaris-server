// multpart upload for box

package box

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ncw/rclone/backend/box/api"
	"github.com/ncw/rclone/fs"
	"github.com/ncw/rclone/fs/accounting"
	"github.com/ncw/rclone/lib/rest"
	"github.com/pkg/errors"
)

// createUploadSession creates an upload session for the object
func (o *Object) createUploadSession(leaf, directoryID string, size int64) (response *api.UploadSessionResponse, err error) {
	opts := rest.Opts{
		Method:  "POST",
		Path:    "/files/upload_sessions",
		RootURL: uploadURL,
	}
	request := api.UploadSessionRequest{
		FileSize: size,
	}
	// If object has an ID then it is existing so create a new version
	if o.id != "" {
		opts.Path = "/files/" + o.id + "/upload_sessions"
	} else {
		opts.Path = "/files/upload_sessions"
		request.FolderID = directoryID
		request.FileName = replaceReservedChars(leaf)
	}
	var resp *http.Response
	err = o.fs.pacer.Call(func() (bool, error) {
		resp, err = o.fs.srv.CallJSON(&opts, &request, &response)
		return shouldRetry(resp, err)
	})
	return
}

// sha1Digest produces a digest using sha1 as per RFC3230
func sha1Digest(digest []byte) string {
	return "sha=" + base64.StdEncoding.EncodeToString(digest)
}

// uploadPart uploads a part in an upload session
func (o *Object) uploadPart(SessionID string, offset, totalSize int64, chunk []byte, wrap accounting.WrapFn) (response *api.UploadPartResponse, err error) {
	chunkSize := int64(len(chunk))
	sha1sum := sha1.Sum(chunk)
	opts := rest.Opts{
		Method:        "PUT",
		Path:          "/files/upload_sessions/" + SessionID,
		RootURL:       uploadURL,
		ContentType:   "application/octet-stream",
		ContentLength: &chunkSize,
		ContentRange:  fmt.Sprintf("bytes %d-%d/%d", offset, offset+chunkSize-1, totalSize),
		ExtraHeaders: map[string]string{
			"Digest": sha1Digest(sha1sum[:]),
		},
	}
	var resp *http.Response
	err = o.fs.pacer.Call(func() (bool, error) {
		opts.Body = wrap(bytes.NewReader(chunk))
		resp, err = o.fs.srv.CallJSON(&opts, nil, &response)
		return shouldRetry(resp, err)
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

// commitUpload finishes an upload session
func (o *Object) commitUpload(SessionID string, parts []api.Part, modTime time.Time, sha1sum []byte) (result *api.FolderItems, err error) {
	opts := rest.Opts{
		Method:  "POST",
		Path:    "/files/upload_sessions/" + SessionID + "/commit",
		RootURL: uploadURL,
		ExtraHeaders: map[string]string{
			"Digest": sha1Digest(sha1sum),
		},
	}
	request := api.CommitUpload{
		Parts: parts,
	}
	request.Attributes.ContentModifiedAt = api.Time(modTime)
	request.Attributes.ContentCreatedAt = api.Time(modTime)
	var body []byte
	var resp *http.Response
	// For discussion of this value see:
	// https://github.com/ncw/rclone/issues/2054
	maxTries := o.fs.opt.CommitRetries
	const defaultDelay = 10
	var tries int
outer:
	for tries = 0; tries < maxTries; tries++ {
		err = o.fs.pacer.Call(func() (bool, error) {
			resp, err = o.fs.srv.CallJSON(&opts, &request, nil)
			if err != nil {
				return shouldRetry(resp, err)
			}
			body, err = rest.ReadBody(resp)
			return shouldRetry(resp, err)
		})
		delay := defaultDelay
		var why string
		if err != nil {
			// Sometimes we get 400 Error with
			// parts_mismatch immediately after uploading
			// the last part.  Ignore this error and wait.
			if boxErr, ok := err.(*api.Error); ok && boxErr.Code == "parts_mismatch" {
				why = err.Error()
			} else {
				return nil, err
			}
		} else {
			switch resp.StatusCode {
			case http.StatusOK, http.StatusCreated:
				break outer
			case http.StatusAccepted:
				why = "not ready yet"
				delayString := resp.Header.Get("Retry-After")
				if delayString != "" {
					delay, err = strconv.Atoi(delayString)
					if err != nil {
						fs.Debugf(o, "Couldn't decode Retry-After header %q: %v", delayString, err)
						delay = defaultDelay
					}
				}
			default:
				return nil, errors.Errorf("unknown HTTP status return %q (%d)", resp.Status, resp.StatusCode)
			}
		}
		fs.Debugf(o, "commit multipart upload failed %d/%d - trying again in %d seconds (%s)", tries+1, maxTries, delay, why)
		time.Sleep(time.Duration(delay) * time.Second)
	}
	if tries >= maxTries {
		return nil, errors.New("too many tries to commit multipart upload - increase --low-level-retries")
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't decode commit response: %q", body)
	}
	return result, nil
}

// abortUpload cancels an upload session
func (o *Object) abortUpload(SessionID string) (err error) {
	opts := rest.Opts{
		Method:     "DELETE",
		Path:       "/files/upload_sessions/" + SessionID,
		RootURL:    uploadURL,
		NoResponse: true,
	}
	var resp *http.Response
	err = o.fs.pacer.Call(func() (bool, error) {
		resp, err = o.fs.srv.Call(&opts)
		return shouldRetry(resp, err)
	})
	return err
}

// uploadMultipart uploads a file using multipart upload
func (o *Object) uploadMultipart(in io.Reader, leaf, directoryID string, size int64, modTime time.Time) (err error) {
	// Create upload session
	session, err := o.createUploadSession(leaf, directoryID, size)
	if err != nil {
		return errors.Wrap(err, "multipart upload create session failed")
	}
	chunkSize := session.PartSize
	fs.Debugf(o, "Multipart upload session started for %d parts of size %v", session.TotalParts, fs.SizeSuffix(chunkSize))

	// Cancel the session if something went wrong
	defer func() {
		if err != nil {
			fs.Debugf(o, "Cancelling multipart upload: %v", err)
			cancelErr := o.abortUpload(session.ID)
			if cancelErr != nil {
				fs.Logf(o, "Failed to cancel multipart upload: %v", err)
			}
		}
	}()

	// unwrap the accounting from the input, we use wrap to put it
	// back on after the buffering
	in, wrap := accounting.UnWrap(in)

	// Upload the chunks
	remaining := size
	position := int64(0)
	parts := make([]api.Part, session.TotalParts)
	hash := sha1.New()
	errs := make(chan error, 1)
	var wg sync.WaitGroup
outer:
	for part := 0; part < session.TotalParts; part++ {
		// Check any errors
		select {
		case err = <-errs:
			break outer
		default:
		}

		reqSize := remaining
		if reqSize >= chunkSize {
			reqSize = chunkSize
		}

		// Make a block of memory
		buf := make([]byte, reqSize)

		// Read the chunk
		_, err = io.ReadFull(in, buf)
		if err != nil {
			err = errors.Wrap(err, "multipart upload failed to read source")
			break outer
		}

		// Make the global hash (must be done sequentially)
		_, _ = hash.Write(buf)

		// Transfer the chunk
		wg.Add(1)
		o.fs.uploadToken.Get()
		go func(part int, position int64) {
			defer wg.Done()
			defer o.fs.uploadToken.Put()
			fs.Debugf(o, "Uploading part %d/%d offset %v/%v part size %v", part+1, session.TotalParts, fs.SizeSuffix(position), fs.SizeSuffix(size), fs.SizeSuffix(chunkSize))
			partResponse, err := o.uploadPart(session.ID, position, size, buf, wrap)
			if err != nil {
				err = errors.Wrap(err, "multipart upload failed to upload part")
				select {
				case errs <- err:
				default:
				}
				return
			}
			parts[part] = partResponse.Part
		}(part, position)

		// ready for next block
		remaining -= chunkSize
		position += chunkSize
	}
	wg.Wait()
	if err == nil {
		select {
		case err = <-errs:
		default:
		}
	}
	if err != nil {
		return err
	}

	// Finalise the upload session
	result, err := o.commitUpload(session.ID, parts, modTime, hash.Sum(nil))
	if err != nil {
		return errors.Wrap(err, "multipart upload failed to finalize")
	}

	if result.TotalCount != 1 || len(result.Entries) != 1 {
		return errors.Errorf("multipart upload failed %v - not sure why", o)
	}
	return o.setMetaData(&result.Entries[0])
}
