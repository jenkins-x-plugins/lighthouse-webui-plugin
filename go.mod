module github.com/jenkins-x-plugins/lighthouse-webui-plugin

require (
	github.com/Jeffail/gabs/v2 v2.6.1
	github.com/Masterminds/goutils v1.1.1
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/RoaringBitmap/roaring v0.5.5 // indirect
	github.com/blevesearch/bleve v1.0.14
	github.com/glycerine/go-unsnap-stream v0.0.0-20210130063903-47dfef350d96 // indirect
	github.com/golang/snappy v0.0.3 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/jenkins-x-plugins/jx-pipeline v0.0.139 // indirect
	github.com/jenkins-x/go-scm v1.10.10
	github.com/jenkins-x/jx-helpers/v3 v3.0.130 // indirect
	github.com/jenkins-x/lighthouse v1.0.44
	github.com/magefile/mage v1.10.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/rickb777/date v1.13.0
	github.com/rs/xid v1.2.1 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/subchord/go-sse v1.0.1 // indirect
	github.com/tektoncd/pipeline v0.20.0 // indirect
	github.com/tinylib/msgp v1.1.5 // indirect
	github.com/unrolled/render v1.0.3
	github.com/urfave/negroni/v2 v2.0.2
	github.com/willf/bitset v1.1.11 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apimachinery v0.21.0
	k8s.io/cli-runtime v0.21.0
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
)

replace (
	// helm dependencies (from jx-pipeline)
	github.com/docker/distribution => github.com/docker/distribution v0.0.0-20191216044856-a8371794149d
	github.com/docker/docker => github.com/moby/moby v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
	// override the go-scm from tekton
	github.com/jenkins-x/go-scm => github.com/jenkins-x/go-scm v1.9.0
	github.com/tektoncd/pipeline => github.com/jenkins-x/pipeline v0.3.2-0.20210118090417-1e821d85abf6
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.20.1
	k8s.io/client-go => k8s.io/client-go v0.20.1
)

go 1.15
