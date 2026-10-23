module github.com/coderyw/gomicro-util

go 1.25.5

require github.com/go-test/deep v1.1.1

require (
	github.com/DataDog/dd-trace-go/contrib/aws/aws-sdk-go-v2/v2 v2.6.0
	github.com/DataDog/dd-trace-go/contrib/aws/aws-sdk-go/v2 v2.6.0
	github.com/DataDog/dd-trace-go/contrib/gin-gonic/gin/v2 v2.6.0
	github.com/DataDog/dd-trace-go/contrib/go-redis/redis/v2 v2.6.0
	github.com/DataDog/dd-trace-go/contrib/google.golang.org/grpc/v2 v2.6.0
	github.com/DataDog/dd-trace-go/contrib/gorm.io/gorm.v1/v2 v2.6.0
	github.com/DataDog/dd-trace-go/contrib/net/http/v2 v2.6.0
	github.com/DataDog/dd-trace-go/contrib/redis/go-redis.v9/v2 v2.6.0
	github.com/DataDog/dd-trace-go/v2 v2.6.0
	github.com/DataDog/orchestrion v1.8.0
	github.com/alicebob/miniredis/v2 v2.38.0
	github.com/aws/aws-sdk-go-v2 v1.45.1
	github.com/aws/aws-sdk-go-v2/config v1.32.7
	github.com/aws/aws-sdk-go-v2/credentials v1.19.7
	github.com/aws/aws-sdk-go-v2/service/s3 v1.89.1
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.47.0
	github.com/aws/smithy-go v1.28.1
	github.com/confluentinc/confluent-kafka-go/v2 v2.4.0
	github.com/go-micro/plugins/v4/transport/grpc v1.2.0
	github.com/go-redis/redis_rate/v10 v10.0.1
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/huaweicloud/huaweicloud-sdk-go-obs v3.25.4+incompatible
	github.com/huaweicloud/huaweicloud-sdk-go-v3 v0.1.214
	github.com/pingcap/tidb/pkg/parser v0.0.0-20260312025557-ea9094d4425b
	github.com/rabbitmq/amqp091-go v1.13.0
	github.com/stathat/consistent v1.0.0
	github.com/ulule/limiter/v3 v3.11.2
	github.com/wneessen/go-mail v0.7.1
	github.com/xuri/excelize/v2 v2.9.0
	go.uber.org/automaxprocs v1.6.0
	gorm.io/driver/mysql v1.6.0
	gorm.io/driver/postgres v1.6.0
	gorm.io/gorm v1.31.1
	gorm.io/plugin/dbresolver v1.6.2
)

require (
	dario.cat/mergo v1.0.2 // indirect
	github.com/DataDog/datadog-agent/comp/core/tagger/origindetection v0.71.2 // indirect
	github.com/DataDog/datadog-agent/pkg/obfuscate v0.71.2 // indirect
	github.com/DataDog/datadog-agent/pkg/opentelemetry-mapping-go/otlp/attributes v0.71.2 // indirect
	github.com/DataDog/datadog-agent/pkg/proto v0.71.2 // indirect
	github.com/DataDog/datadog-agent/pkg/remoteconfig/state v0.73.0-rc.1 // indirect
	github.com/DataDog/datadog-agent/pkg/trace v0.71.2 // indirect
	github.com/DataDog/datadog-agent/pkg/util/log v0.71.2 // indirect
	github.com/DataDog/datadog-agent/pkg/util/scrubber v0.71.2 // indirect
	github.com/DataDog/datadog-agent/pkg/version v0.71.2 // indirect
	github.com/DataDog/datadog-go/v5 v5.8.1 // indirect
	github.com/DataDog/go-libddwaf/v4 v4.8.0 // indirect
	github.com/DataDog/go-runtime-metrics-internal v0.0.4-0.20250806100345-ca5e7fdaf7b6 // indirect
	github.com/DataDog/go-sqllexer v0.1.9 // indirect
	github.com/DataDog/go-tuf v1.1.1-0.5.2 // indirect
	github.com/DataDog/sketches-go v1.4.7 // indirect
	github.com/HdrHistogram/hdrhistogram-go v1.1.2 // indirect
	github.com/alibabacloud-go/alibabacloud-gateway-pop v0.0.6 // indirect
	github.com/alibabacloud-go/darabonba-array v0.1.0 // indirect
	github.com/alibabacloud-go/darabonba-encode-util v0.0.2 // indirect
	github.com/alibabacloud-go/darabonba-map v0.0.2 // indirect
	github.com/alibabacloud-go/darabonba-signature-util v0.0.7 // indirect
	github.com/alibabacloud-go/darabonba-string v1.0.2 // indirect
	github.com/alibabacloud-go/kms-20160120/v3 v3.2.3 // indirect
	github.com/aliyun/aliyun-secretsmanager-client-go v1.1.5 // indirect
	github.com/antithesishq/antithesis-sdk-go v0.5.0 // indirect
	github.com/aws/aws-sdk-go v1.44.327 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.4 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.17 // indirect
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.17.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.31.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/eventbridge v1.30.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.9.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/kinesis v1.27.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sfn v1.26.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sns v1.29.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sqs v1.31.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.6 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/blakesmith/ar v0.0.0-20190502131153-809d4375e1fb // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/charmbracelet/colorprofile v0.3.2 // indirect
	github.com/charmbracelet/lipgloss v1.1.0 // indirect
	github.com/charmbracelet/x/ansi v0.10.2 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.13 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/cihub/seelog v0.0.0-20170130134532-f561c5e57575 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.3.0 // indirect
	github.com/compose-spec/compose-go/v2 v2.9.1 // indirect
	github.com/containerd/containerd v1.7.29 // indirect
	github.com/containerd/platforms v1.0.0-rc.2 // indirect
	github.com/cyphar/filepath-securejoin v0.6.0 // indirect
	github.com/dave/dst v0.27.3 // indirect
	github.com/deckarep/golang-set v1.7.1 // indirect
	github.com/docker/buildx v0.32.1 // indirect
	github.com/docker/cli v29.2.1+incompatible // indirect
	github.com/docker/compose/v2 v2.40.3 // indirect
	github.com/docker/docker v28.5.2+incompatible // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ebitengine/purego v0.9.0 // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-acme/lego/v4 v4.4.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-redis/redis v6.15.9+incompatible // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.0.4 // indirect
	github.com/goccy/go-yaml v1.18.0 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/google/go-tpm v0.9.6 // indirect
	github.com/google/pprof v0.0.0-20250820193118-f64d9cf942d6 // indirect
	github.com/gorilla/handlers v1.5.1 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/linkdata/deadlock v0.5.5 // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/minio/highwayhash v1.0.3 // indirect
	github.com/minio/simdjson-go v0.4.5 // indirect
	github.com/moby/buildkit v0.28.0 // indirect
	github.com/moby/go-archive v0.2.0 // indirect
	github.com/moby/moby/api v1.53.0 // indirect
	github.com/moby/moby/client v0.2.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/morikuni/aec v1.1.0 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nats-io/jwt/v2 v2.8.0 // indirect
	github.com/nats-io/nats-server/v2 v2.12.1 // indirect
	github.com/nats-io/nats.go v1.47.0 // indirect
	github.com/nats-io/nkeys v0.4.11 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/orcaman/concurrent-map v0.0.0-20210501183033-44dafcb38ecc // indirect
	github.com/outcaste-io/ristretto v0.2.3 // indirect
	github.com/petermattis/goid v0.0.0-20250813065127-a731cc31b4fe // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/pingcap/errors v0.11.5-0.20250523034308-74f78ae071ee // indirect
	github.com/pingcap/failpoint v0.0.0-20240528011301-b51a646c7c86 // indirect
	github.com/pingcap/log v1.1.0 // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/polyfloyd/go-errorlint v1.8.1-0.20250906200200-9b25878c4dea // indirect
	github.com/puzpuzpuz/xsync/v3 v3.5.1 // indirect
	github.com/richardlehane/mscfb v1.0.4 // indirect
	github.com/richardlehane/msoleps v1.0.4 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	github.com/secure-systems-lab/go-securesystemslib v0.10.0 // indirect
	github.com/shirou/gopsutil/v4 v4.25.10 // indirect
	github.com/shoenig/go-m1cpu v0.1.7 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/skeema/knownhosts v1.3.0 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/testcontainers/testcontainers-go v0.30.0 // indirect
	github.com/tinylib/msgp v1.5.0 // indirect
	github.com/trailofbits/go-mutexasserts v0.0.0-20250514102930-c1f3d2e37561 // indirect
	github.com/vektah/gqlparser/v2 v2.5.31 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/xuri/efp v0.0.0-20240408161823-9ad904a10d6d // indirect
	github.com/xuri/nfp v0.0.0-20240318013403-ab9948c2c4a7 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.mongodb.org/mongo-driver v1.17.6 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/collector/component v1.44.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.46.0 // indirect
	go.opentelemetry.io/collector/internal/telemetry v0.138.0 // indirect
	go.opentelemetry.io/collector/pdata v1.46.0 // indirect
	go.opentelemetry.io/collector/pdata/pprofile v0.140.0 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.13.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.63.0 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.38.0 // indirect
	go.opentelemetry.io/otel/log v0.14.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/term v0.40.0 // indirect
	k8s.io/apimachinery v0.34.1 // indirect
	stathat.com/c/consistent v1.0.0 // indirect
	tags.cncf.io/container-device-interface v1.1.0 // indirect
)

// gorm fork project
require github.com/coderyw/protobuf v1.6.5

//replace (
//	github.com/coderyw/gorm => gorm.io/gorm v1.31.1
//	github.com/coderyw/gorm-driver-mysql => gorm.io/driver/mysql v1.6.0
//	github.com/coderyw/gorm-plugin-dbresolver => gorm.io/plugin/dbresolver v1.6.2
//	github.com/coderyw/postgres => gorm.io/driver/postgres v1.6.0
//)

//replace github.com/coderyw/easyjson => ../../../mygithub/easyjson

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	gitee.com/opengauss/openGauss-connector-go-pq v1.0.4
	github.com/IBM/sarama v1.43.2
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/ProtonMail/go-crypto v1.3.0 // indirect
	github.com/alibabacloud-go/alibabacloud-gateway-spi v0.0.5 // indirect
	github.com/alibabacloud-go/darabonba-openapi/v2 v2.0.10
	github.com/alibabacloud-go/debug v1.0.1 // indirect
	github.com/alibabacloud-go/dysmsapi-20170525/v3 v3.0.6
	github.com/alibabacloud-go/endpoint-util v1.1.0 // indirect
	github.com/alibabacloud-go/openapi-util v0.1.0 // indirect
	github.com/alibabacloud-go/tea v1.2.2
	github.com/alibabacloud-go/tea-utils v1.4.4 // indirect
	github.com/alibabacloud-go/tea-utils/v2 v2.0.7
	github.com/alibabacloud-go/tea-xml v1.1.3 // indirect
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.1800 // indirect
	github.com/aliyun/alibabacloud-dkms-gcs-go-sdk v0.5.1 // indirect
	github.com/aliyun/alibabacloud-dkms-transfer-go-sdk v0.1.8 // indirect
	github.com/aliyun/credentials-go v1.4.3 // indirect
	github.com/apache/dubbo-getty v1.4.9-0.20220825024508-3da63c3257fa // indirect
	github.com/apache/skywalking-go/toolkit v0.6.0
	github.com/benbjohnson/clock v1.3.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bitly/go-simplejson v0.5.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/bwmarrin/snowflake v0.3.0
	github.com/bytedance/sonic v1.12.4 // indirect
	github.com/bytedance/sonic/loader v0.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/clbanning/mxj/v2 v2.5.5 // indirect
	github.com/cloudflare/circl v1.6.3 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/coderyw/easyjson v0.0.1
	github.com/coderyw/easymap v0.3.7
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.7.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dubbogo/gost v1.12.6-0.20220824084206-300e27e9e524 // indirect
	github.com/eapache/go-resiliency v1.6.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20230731223053-c322873962e3 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.6 // indirect
	github.com/gin-contrib/pprof v1.5.1
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/gin-gonic/gin v1.10.1
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.6.0 // indirect
	github.com/go-git/go-git/v5 v5.13.0 // indirect
	github.com/go-micro/plugins/v4/registry/kubernetes v1.1.2
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.22.1 // indirect
	github.com/go-sql-driver/mysql v1.8.1
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/mock v1.7.0-rc.1 // indirect
	github.com/golang/protobuf v1.5.4
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.6.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.4 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/jinzhu/copier v0.3.5
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.13-0.20220915233716-71ac16282d12
	github.com/juju/ratelimit v1.0.2
	github.com/k0kubun/pp v3.0.1+incompatible // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/compress v1.18.4 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/longbridgeapp/assert v1.1.0
	github.com/longbridgeapp/sqlparser v0.3.2
	github.com/lufia/plan9stats v0.0.0-20251013123823-9fd1530e3ec3 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/miekg/dns v1.1.50 // indirect
	github.com/mitchellh/hashstructure v1.1.0
	github.com/mitchellh/mapstructure v1.5.1-0.20231216201459-8508981c8b6c
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/nacos-group/nacos-sdk-go/v2 v2.3.2
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/opentracing/opentracing-go v1.2.0
	github.com/oxtoacart/bpool v0.0.0-20190530202638-03653db5a59c // indirect
	github.com/panjf2000/ants/v2 v2.9.1
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/prometheus/client_golang v1.23.2
	github.com/prometheus/client_model v0.6.2
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20250401214520-65e299d6c5c9 // indirect
	github.com/redis/go-redis/v9 v9.20.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/seata/seata-go v1.2.0
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/shirou/gopsutil/v3 v3.24.4 // indirect
	github.com/shopspring/decimal v1.4.0
	github.com/stretchr/testify v1.11.1
	github.com/tjfoc/gmsm v1.4.1 // indirect
	github.com/tklauser/go-sysconf v0.3.15 // indirect
	github.com/tklauser/numcpus v0.10.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/uber/jaeger-client-go v2.30.0+incompatible
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	github.com/urfave/cli/v2 v2.27.7
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/xrash/smetrics v0.0.0-20250705151800-55b8f293f342 // indirect
	github.com/xwb1989/sqlparser v0.0.0-20180606152119-120387863bf2
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	gitlab.jkluck.vip/base/eonid/eonid-go/v2 v2.1.0-rc.2
	go-micro.dev/v4 v4.11.0
	go.etcd.io/etcd/api/v3 v3.5.14
	go.etcd.io/etcd/client/pkg/v3 v3.5.14 // indirect
	go.etcd.io/etcd/client/v3 v3.5.14
	go.uber.org/atomic v1.11.0
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/ratelimit v0.3.1
	go.uber.org/zap v1.27.0
	golang.org/x/arch v0.12.0 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/exp v0.0.0-20251023183803-a4bb9ffd2546
	golang.org/x/mod v0.33.0 // indirect
	golang.org/x/net v0.51.0
	golang.org/x/sync v0.19.0
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	golang.org/x/tools v0.41.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251222181119-0a764e51fe1b // indirect
	google.golang.org/grpc v1.78.0
	google.golang.org/grpc/examples v0.0.0-20240701171118-c9caa9ed53d7
	google.golang.org/protobuf v1.36.11
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.1
	vimagination.zapto.org/byteio v0.0.0-20200222190125-d27cba0f0b10 // indirect
)
