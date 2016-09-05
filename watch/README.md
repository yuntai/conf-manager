# Testing wait feature using Curl

### put key first
curl -v -X PUT -d 'test' http://localhost:8500/v1/kv/web/key1
[{"LockIndex":0,"Key":"web/key1","Flags":0,"Value":"dGVzdA==","CreateIndex":27,"ModifyIndex":44}]

### wait for the value changed using the ModifyIndex
curl -v -X GET http://localhost:8500/v1/kv/web/key1?index=41
