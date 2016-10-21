# running consul agent

# consul agent -dev -bind=0.0.0.0 -data-dir=/mnt/tmp/consul-data-dir -advertise=127.0.0.1


# TEST GIT REPO
TESTGITROOT=/mnt/tmp/testapp
rm -rf $TESTGITROOT
mkdir -p $TESTGITROOT 

pushd .
cd $TESTGITROOT
git init
echo "some config" >> README
git add README
git ci -am "initial commit"
popd

# populate global kv space
appId=testapp
kv=localhost:8500/v1/kv/config/global/$appId

curl -X PUT -d @- $kv/id <<< $appId
curl -X PUT -d @- $kv/branch <<< master
curl -X PUT -d @- $kv/repo <<< "file://$TESTGITROOT"
curl -X PUT -d @- $kv/rev <<< latest

# check ui
# http://localhost:8500/ui/

# /Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome http://localhost:8080 http://localhost:8500/ui/

# run simple app under consul-template under supervisory
consul-template -consul localhost:8500 -template "testapp.ctmpl:testapp.json" -exec="main" -log-level=info -exec-reload-signal=SIGHUP

################################
# service example
#curl -X PUT -d @service.json localhost:8500/v1/agent/service/register

# run consul template
#consul-template -consul localhost:8500 -template "testapp.ctmpl:testapp.json"

# initialize global space for KV stroage
# https://github.com/JoergM/consul-examples/tree/master/http_api

