module github.com/otelwasm/otelwasm/examples

go 1.24.2

require (
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor v0.125.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/awss3receiver v0.125.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/webhookeventreceiver v0.125.0
	github.com/otelwasm/otelwasm/guest v0.0.0
	go.opentelemetry.io/collector/component v1.32.0
	go.opentelemetry.io/collector/component/componenttest v0.126.0
	go.opentelemetry.io/collector/exporter v0.126.0
	go.opentelemetry.io/collector/pdata v1.32.0
	go.opentelemetry.io/collector/processor v1.32.0
	go.opentelemetry.io/collector/receiver v1.32.0
	go.uber.org/zap v1.27.0
)

require (
	github.com/alecthomas/participle/v2 v2.1.4 // indirect
	github.com/antchfx/xmlquery v1.4.4 // indirect
	github.com/antchfx/xpath v1.3.4 // indirect
	github.com/cenkalti/backoff/v5 v5.0.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/elastic/go-grok v0.3.1 // indirect
	github.com/elastic/lunes v0.1.0 // indirect
	github.com/expr-lang/expr v1.17.2 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/itchyny/timefmt-go v0.1.6 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/knadh/koanf/providers/confmap v1.0.0 // indirect
	github.com/knadh/koanf/v2 v2.2.0 // indirect
	github.com/magefile/mage v1.15.0 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/mostynb/go-grpc-compression v1.2.3 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal v0.125.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter v0.125.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/batchperresourceattr v0.126.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl v0.125.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatautil v0.125.0 // indirect
	github.com/tilinna/clock v1.1.0 // indirect
	github.com/twmb/murmur3 v1.1.8 // indirect
	github.com/ua-parser/uap-go v0.0.0-20240611065828-3a4781585db6 // indirect
	go.opentelemetry.io/collector v0.125.0 // indirect
	go.opentelemetry.io/collector/config/configgrpc v0.125.0 // indirect
	go.opentelemetry.io/collector/config/confignet v1.31.0 // indirect
	go.opentelemetry.io/collector/config/configretry v1.32.0 // indirect
	go.opentelemetry.io/collector/confmap v1.32.0 // indirect
	go.opentelemetry.io/collector/consumer/consumererror v0.126.0 // indirect
	go.opentelemetry.io/collector/consumer/consumererror/xconsumererror v0.125.0 // indirect
	go.opentelemetry.io/collector/consumer/xconsumer v0.126.0 // indirect
	go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper v0.125.0 // indirect
	go.opentelemetry.io/collector/exporter/xexporter v0.126.0 // indirect
	go.opentelemetry.io/collector/extension v1.32.0 // indirect
	go.opentelemetry.io/collector/extension/xextension v0.126.0 // indirect
	go.opentelemetry.io/collector/internal/sharedcomponent v0.125.0 // indirect
	go.opentelemetry.io/collector/pdata/pprofile v0.126.0 // indirect
	go.opentelemetry.io/collector/pipeline/xpipeline v0.126.0 // indirect
	go.opentelemetry.io/collector/processor/processorhelper v0.125.0 // indirect
	go.opentelemetry.io/collector/receiver/xreceiver v0.126.0 // indirect
	go.opentelemetry.io/collector/semconv v0.125.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.60.0 // indirect
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

require (
	github.com/aws/aws-sdk-go-v2 v1.36.3 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.10 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.29.14 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.67 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.30 // indirect
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.17.75 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.7.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.18.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.79.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.25.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.30.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.19 // indirect
	github.com/aws/smithy-go v1.22.2 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/julienschmidt/httprouter v1.3.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/open-telemetry/opamp-go v0.19.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awss3exporter v0.126.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/opampcustommessages v0.125.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/rs/cors v1.11.1 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/collector/client v1.32.0 // indirect
	go.opentelemetry.io/collector/component/componentstatus v0.126.0 // indirect
	go.opentelemetry.io/collector/config/configauth v0.125.0 // indirect
	go.opentelemetry.io/collector/config/configcompression v1.32.0 // indirect
	go.opentelemetry.io/collector/config/confighttp v0.125.0 // indirect
	go.opentelemetry.io/collector/config/configmiddleware v0.125.0 // indirect
	go.opentelemetry.io/collector/config/configopaque v1.31.0 // indirect
	go.opentelemetry.io/collector/config/configtls v1.31.0 // indirect
	go.opentelemetry.io/collector/consumer v1.32.0 // indirect
	go.opentelemetry.io/collector/exporter/otlphttpexporter v0.125.0
	go.opentelemetry.io/collector/extension/extensionauth v1.31.0 // indirect
	go.opentelemetry.io/collector/extension/extensionmiddleware v0.125.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.32.0 // indirect
	go.opentelemetry.io/collector/internal/telemetry v0.126.0 // indirect
	go.opentelemetry.io/collector/pipeline v0.126.0 // indirect
	go.opentelemetry.io/collector/receiver/otlpreceiver v0.125.0
	go.opentelemetry.io/collector/receiver/receiverhelper v0.125.0 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.10.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.60.0 // indirect
	go.opentelemetry.io/otel v1.35.0 // indirect
	go.opentelemetry.io/otel/log v0.11.0 // indirect
	go.opentelemetry.io/otel/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/sdk v1.35.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
	google.golang.org/grpc v1.72.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)

replace github.com/otelwasm/otelwasm/guest => ../guest
