package pod

import (
	"context"
	 "time"
	 "bytes"
	 "io"
	 "os"
	 "strings"
	corev1 "k8s.io/api/core/v1"
	"github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
	//"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	// "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/apimachinery/pkg/runtime"
	// "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	// "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_pod")


// Add creates a new PodLog Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcilePod {client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("pod-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Pod
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcilePod {}

// ReconcilePod  reconciles a Pod object
type ReconcilePod struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}


// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcilePod ) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	instance := &corev1.Pod{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger1 := log.WithValues("namespace", request.Namespace, "name", request.Name)
			reqLogger1.Info("pod deleted")
			return reconcile.Result{}, nil
		}	
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	} else {
		if instance.Status.Phase == "Running" && instance.Status.Conditions[1].Status == "True" {
			podLogs := make(map[string]string)
			for i:=0 ; i < len(instance.Spec.Containers) ; i++ {
				containerName := instance.Spec.Containers[i].Name
			    clogs, err := getPodLogs(containerName,request)
			    if err != nil {
				   reqLogger := log.WithValues("error", err)
				   reqLogger.Info("error while fetching logs")
				} else {
                   podLogs[containerName] = clogs
				}
			}
			time.Sleep(5 * time.Second)
			err = r.client.Get(context.TODO(), request.NamespacedName, instance)
			  if err == nil && instance.Status.Phase == "Running" && instance.Status.Conditions[1].Status == "True" {
			    reqLogger := log.WithValues("namespace", request.Namespace, "name", request.Name)
				reqLogger.Info("pod created")
			   } else {
				for container, clog := range(podLogs) {
				  err := dumpLogsToS3(request,container,clog)
				  if err != nil {
					reqLogger := log.WithValues("error", err)
					reqLogger.Info("error while putting logs to s3")
				  } 
		        }
			}
		}		
	}	
	return reconcile.Result{}, nil
	
}
func getPodLogs(containerName string,request reconcile.Request) (string, error) {			
	  config, err := rest.InClusterConfig()
	  if err != nil {
	  panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
	  panic(err.Error())
	}
	req := clientset.CoreV1().Pods(request.Namespace).GetLogs(request.Name, &corev1.PodLogOptions{Container: containerName })
	podLogs, err := req.Stream()
	if err != nil {
		return "error in opening stream", err
	}
	defer podLogs.Close()
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
	    return "error in copy information from podLogs to buffer", err
	}
	str := buf.String()
    return str,nil
}

func dumpLogsToS3(request reconcile.Request, containerName string, podLogs string) error {
  logDumpBucket := os.Getenv("LOG_DUMP_BUCKET")
  region := os.Getenv("AWS_REGION_NAME")
  logfilename := "/"+request.Namespace+"/"+request.Name+"/"+containerName+".log"
  sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(region)}))
  uploader := s3manager.NewUploader(sess)
  _, err := uploader.Upload(&s3manager.UploadInput{
    Bucket: aws.String(logDumpBucket),
    Key:    aws.String(logfilename),
    Body:   strings.NewReader(podLogs),
})
if err != nil {
	return err
  }
  reqLogger := log.WithValues("namespace", request.Namespace, "pod", request.Name, "container", containerName, "s3bucketlocation", logDumpBucket+logfilename)
  reqLogger.Info("log dump info")
  return nil
}


