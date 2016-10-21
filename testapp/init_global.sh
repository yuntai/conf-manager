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
