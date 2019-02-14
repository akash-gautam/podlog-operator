# podlog-operator
A simple operator which logs the creation and deletion of pod across all namespaces in a k8s cluster and puts the logs of all the containers running inside a pod to AWS S3 when the pod is terminated. It also logs the url of the s3 bucket location where logs are dumped the format is `bucket-name/name-space/pod-name/container-name.log`

## Steps to deploy the podlog operator
All the files required to deploy the operator is available in the `deploy` folder of the repository.
1. Deploy the service accout `kubectl apply -f deploy/service_account.yaml`
2. Deploy the role `kubectl apply -f deploy/role.yaml`
3. Deploy the role bindings `kubectl apply -f deploy/role_binding.yaml`
4. Deploy the operator, before deploying the operator we need to update the `operator.yaml` file to provide the name of your
   s3 bucket and the aws region of the bucket, post this update apply the file `kubectl apply -f deploy/operator.yaml`
5. Make sure that the servers on which the operator is running has write permission to the s3 bucket.

## Steps to verify the working of podlog operator
Once the podlog operator is deployed we can verify its working by deploying new pods on the system and tail the logs of the operator pod.
 1. Deploy one nginx instance in the namespace gateway `kubectl run nginx --image=nginx --port=80 -n gateway`
 2. Get the ip of the nginx pod that came up `kubectl describe pod -n gateway | grep IP`
 3. make request to the nginx pod to generate some logs `curl <pod-ip>
 4. Depoy an echo server in the namespace backend `kubectl run echoserver --image=gcr.io/google_containers/echoserver:1.4 --port=8080 -n backend`
 5. Get the ip of the echoserver pod that came up `kubectl describe pod -n backend | grep IP`
 6. make request to the echoserver pod to generate some logs `curl <pod-ip>:8080/echo`
 7. Delete the echo server deployment `kubectl delete deployment echoserver -n backend`
 8. Delete the nginx server deployment `kubectl delete deployment nginx -n gateway`
 Above operatios will result in the following type of logs on the operator 
 ```{"level":"info","ts":1550140421.0137541,"logger":"controller_pod","msg":"pod created","namespace":"gateway","name":"nginx-57867cc648-vz4q8"}
{"level":"info","ts":1550140537.2742467,"logger":"controller_pod","msg":"pod created","namespace":"backend","name":"echoserver-6c59bf6c9-m5tmv"}
{"level":"info","ts":1550140670.0213962,"logger":"controller_pod","msg":"log dump info","namespace":"backend","pod":"echoserver-6c59bf6c9-m5tmv","container":"echoserver","s3bucketlocation":"podlogdumpbucket/backend/echoserver-6c59bf6c9-m5tmv/echoserver.log"}
{"level":"info","ts":1550140671.043203,"logger":"controller_pod","msg":"pod deleted","namespace":"backend","name":"echoserver-6c59bf6c9-m5tmv"}
{"level":"info","ts":1550140803.2885888,"logger":"controller_pod","msg":"log dump info","namespace":"gateway","pod":"nginx-57867cc648-vz4q8","container":"nginx","s3bucketlocation":"podlogdumpbucket/gateway/nginx-57867cc648-vz4q8/nginx.log"}
{"level":"info","ts":1550140803.288894,"logger":"controller_pod","msg":"pod deleted","namespace":"gateway","name":"nginx-57867cc648-vz4q8"} 
```
9. Verify on the AWS coonsole that the s3 bucket has log file in the format <namespace>/pod-name/container-name.log

## Steps to build the podlog conatiner
If you have made changes to the code then you can follow the following steps to build the image
1. Install and configure operator sdk. As explaned here https://github.com/operator-framework/operator-sdk/blob/master/doc/user-guide.md
2. From the root folder run `operator-sdk build <tag for the image>`



