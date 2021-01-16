module gitlab.com/olaris/olaris-server

go 1.12

replace (
	github.com/elazarl/go-bindata-assetfs => github.com/elazarl/go-bindata-assetfs v1.0.1-0.20191027195357-d0111fe6fb11
	github.com/pkg/sftp => github.com/pkg/sftp v1.10.0
	github.com/rfjakob/eme => github.com/rfjakob/eme v0.0.0-20170305125520-01668ae55fe0
	github.com/yunify/qingstor-sdk-go => github.com/yunify/qingstor-sdk-go v0.0.0-20190425063759-60a6f6383677
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20190513172903-22d7a77e9e5f
	golang.org/x/net => golang.org/x/net v0.0.0-20190522155817-f3200d17e092
	golang.org/x/sys => golang.org/x/sys v0.0.0-20190522044717-8097e1b27ff5
)

require (
	github.com/Jeffail/tunny v0.0.0-20181108205650-4921fff29480
	github.com/coreos/bbolt v1.3.3 // indirect
	github.com/coreos/go-etcd v2.0.0+incompatible // indirect
	github.com/cpuguy83/go-md2man v1.0.10 // indirect
	github.com/dgrijalva/jwt-go v3.2.1-0.20180921172315-3af4c746e1c2+incompatible
	github.com/elazarl/go-bindata v3.0.5+incompatible // indirect
	github.com/elazarl/go-bindata-assetfs v1.0.1
	github.com/fsnotify/fsnotify v1.4.7
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/goava/di v1.2.0
	github.com/google/uuid v1.1.1
	github.com/gorilla/handlers v1.4.0
	github.com/gorilla/mux v1.7.2
	github.com/grandcat/zeroconf v1.0.0
	github.com/graph-gophers/graphql-go v0.0.0-20190513003547-158e7b876106
	github.com/graph-gophers/graphql-transport-ws v0.0.0-20190611222414-40c048432299
	github.com/jinzhu/gorm v1.9.9-0.20190611093255-321c636b9da5
	github.com/kr/fs v0.1.0 // indirect
	github.com/kylelemons/go-gypsy v0.0.0-20160905020020-08cad365cd28 // indirect
	github.com/maxbrunsfeld/counterfeiter/v6 v6.2.3
	github.com/opentracing/opentracing-go v1.1.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/rclone/rclone v1.52.3
	github.com/rs/cors v1.6.0
	github.com/ryanbradynd05/go-tmdb v0.0.0-20181220020137-291a20d25ffd
	github.com/satori/go.uuid v1.2.1-0.20181028125025-b2ce2384e17b
	github.com/sirupsen/logrus v1.4.2
	github.com/snowzach/rotatefilehook v0.0.0-20180327172521-2f64f265f58c
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.1
	github.com/stretchr/testify v1.5.1
	github.com/ugorji/go/codec v0.0.0-20181204163529-d75b2dcb6bc8 // indirect
	golang.org/x/mod v0.3.0 // indirect
	golang.org/x/sys v0.0.0-20200323222414-85ca7c5b95cd
	golang.org/x/tools v0.0.0-20200529172331-a64b76657301 // indirect
	gopkg.in/gormigrate.v1 v1.5.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0-20170531160350-a96e63847dc3 // indirect
)
