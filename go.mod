module gitlab.com/olaris/olaris-server

go 1.18

replace github.com/jlaffaye/ftp => github.com/rclone/ftp v1.0.0-210902f

require (
	github.com/Jeffail/tunny v0.1.4
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/elazarl/go-bindata-assetfs v1.0.1
	github.com/fsnotify/fsnotify v1.5.1
	github.com/goava/di v1.11.1
	github.com/google/uuid v1.3.0
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/grandcat/zeroconf v1.0.0
	github.com/graph-gophers/graphql-go v1.3.0
	github.com/graph-gophers/graphql-transport-ws v0.0.1
	github.com/jinzhu/gorm v1.9.16
	github.com/maxbrunsfeld/counterfeiter/v6 v6.4.1
	github.com/pkg/errors v0.9.1
	github.com/rclone/rclone v1.57.0
	github.com/rs/cors v1.8.2
	github.com/ryanbradynd05/go-tmdb v0.0.0-20201006144520-c0566c3d1506
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/snowzach/rotatefilehook v0.0.0-20180327172521-2f64f265f58c
	github.com/spf13/cobra v1.3.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/sys v0.0.0-20211210111614-af8b64212486
	gopkg.in/gormigrate.v1 v1.6.0
)

require (
	cloud.google.com/go v0.99.0 // indirect
	github.com/Azure/azure-pipeline-go v0.2.3 // indirect
	github.com/Azure/azure-storage-blob-go v0.14.0 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.14 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/Azure/go-ntlmssp v0.0.0-20200615164410-66371956d46c // indirect
	github.com/Microsoft/go-winio v0.5.0 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/Unknwon/goconfig v0.0.0-20200908083735-df7de6a44db8 // indirect
	github.com/abbot/go-http-auth v0.4.0 // indirect
	github.com/aws/aws-sdk-go v1.40.27 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/btcsuite/btcutil v1.0.3-0.20201208143702-a53e38424cce // indirect
	github.com/buengese/sgzip v0.1.1 // indirect
	github.com/calebcase/tmpfile v1.0.3 // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/colinmarc/hdfs/v2 v2.2.0 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dropbox/dropbox-sdk-go-unofficial/v6 v6.0.3 // indirect
	github.com/felixge/httpsnoop v1.0.1 // indirect
	github.com/form3tech-oss/jwt-go v3.2.5+incompatible // indirect
	github.com/gabriel-vasile/mimetype v1.3.1 // indirect
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/go-sql-driver/mysql v1.5.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/googleapis/gax-go/v2 v2.1.1 // indirect
	github.com/gorilla/websocket v1.4.0 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/iguanesolutions/go-systemd/v5 v5.1.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.0.0 // indirect
	github.com/jcmturner/goidentity/v6 v6.0.1 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.2 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jlaffaye/ftp v0.0.0-20210307004419-5d4190119067 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jzelinskie/whirlpool v0.0.0-20201016144138-0675e54bb004 // indirect
	github.com/klauspost/compress v1.13.4 // indirect
	github.com/koofr/go-httpclient v0.0.0-20200420163713-93aa7c75b348 // indirect
	github.com/koofr/go-koofrclient v0.0.0-20190724113126-8e5366da203a // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kylelemons/go-gypsy v1.0.0 // indirect
	github.com/lib/pq v1.1.1 // indirect
	github.com/magiconair/properties v1.8.5 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-ieproxy v0.0.1 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-sqlite3 v1.14.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/miekg/dns v1.1.42 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/ncw/go-acd v0.0.0-20201019170801-fe55f33415b1 // indirect
	github.com/ncw/swift/v2 v2.0.1 // indirect
	github.com/opentracing/opentracing-go v1.1.0 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pelletier/go-toml v1.9.4 // indirect
	github.com/pengsrc/go-shared v0.2.1-0.20190131101655-1999055a4a14 // indirect
	github.com/pkg/sftp v1.13.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.11.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.30.0 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/putdotio/go-putio/putio v0.0.0-20200123120452-16d982cac2b8 // indirect
	github.com/rfjakob/eme v1.1.2 // indirect
	github.com/shirou/gopsutil/v3 v3.21.8 // indirect
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966 // indirect
	github.com/spacemonkeygo/monkit/v3 v3.0.15 // indirect
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/cast v1.4.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/subosito/gotenv v1.2.0 // indirect
	github.com/t3rm1n4l/go-mega v0.0.0-20200416171014-ffad7fcb44b8 // indirect
	github.com/tklauser/go-sysconf v0.3.9 // indirect
	github.com/tklauser/numcpus v0.3.0 // indirect
	github.com/vivint/infectious v0.0.0-20200605153912-25a574ae18a3 // indirect
	github.com/xanzy/ssh-agent v0.3.1 // indirect
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a // indirect
	github.com/yunify/qingstor-sdk-go/v3 v3.2.0 // indirect
	github.com/zeebo/errs v1.2.2 // indirect
	gitlab.com/olaris/olaris-rename v0.0.0-20220506133140-36be76dcbab5 // indirect
	go.etcd.io/bbolt v1.3.6 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.19.0 // indirect
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5 // indirect
	golang.org/x/mod v0.5.0 // indirect
	golang.org/x/net v0.0.0-20210813160813-60bc85c4be6d // indirect
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	golang.org/x/tools v0.1.5 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/api v0.63.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20211208223120-3a66f561d7aa // indirect
	google.golang.org/grpc v1.43.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/ini.v1 v1.66.2 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	storj.io/common v0.0.0-20210818163656-4667d2cafb27 // indirect
	storj.io/drpc v0.0.24 // indirect
	storj.io/uplink v1.4.6 // indirect
)
