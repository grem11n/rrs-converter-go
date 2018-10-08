package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/fatih/color"
	"gopkg.in/cheggaaa/pb.v1"
)

// attrs store user-defined parameters
type attrs struct {
	Region      string
	Bucket      string
	Config      string
	Section     string
	Type        string
	Concurrency int
}

// Get user-defined parameters from CLI
var (
	bucketPtr      = flag.String("bucket", "", "Defines bucket. This is a mandatory paramenter!")
	regionPtr      = flag.String("region", "", "Defines region")
	configPtr      = flag.String("config", "", "Allow changing AWS account")
	sectionPtr     = flag.String("section", "default", "Which part of AWS credentials to use")
	typePtr        = flag.String("type", "STANDARD", "Define AWS storage class to convert to: STANDARD, REDUCED_REDUNDANCY, GLACIER. Default: STANDARD")
	concurrencyPtr = flag.Int("maxcon", 10, "Set up maximum concurrency for this task. Default is 10")
)

func checkClass(storageType string) string {
	var class string
	if storageType == "STANDARD" || storageType == "standard" {
		class = "STANDARD"
	} else if storageType == "REDUCED_REDUNDANCY" || storageType == "reduced_redundancy" {
		class = "REDUCED_REDUNDANCY"
	} else if storageType == "GLACIER" || storageType == "glacier" {
		class = "GLACIER"
	} else {
		log.Fatal("Unknown S3 storage class!")
	}
	return class
}
func logger(bucket string, info map[string]error) {
	f, err := os.Create(bucket + "-error.log")
	if err != nil {
		log.Println("Script ended with some errors, but log-file wasn't created due to: ", err)
	}
	defer f.Close()

	logFile, err := os.OpenFile(bucket+"-error.log", os.O_WRONLY, 0666)
	if err != nil {
		log.Println("Script ended with some errors, but log-file wasn't written due to: ", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)
	for object, warning := range info {
		log.Println("WARNING: Some issues occur while processing ", object, warning)
	}
	endMessg := "Script finished with some errors. Check " + bucket + "-error.log for details"
	color.Red(endMessg)
}

func convert(attrs attrs) map[string]error {
	warns := map[string]error{}
	creds := credentials.NewSharedCredentials(attrs.Config, attrs.Section)
	_, err := creds.Get()
	if err != nil {
		color.Set(color.FgRed)
		log.Fatal(err)
		color.Unset()
	}
	// Create new connection to S3
	svc := s3.New(session.New(), &aws.Config{
		Region:      aws.String(attrs.Region),
		Credentials: creds,
	})
	params := &s3.ListObjectsInput{
		Bucket: aws.String(attrs.Bucket),
	}

	var contents []*s3.Object
	svc.ListObjectsPages(params, func(page *s3.ListObjectsOutput, isLastPage bool) bool {
		for _, o := range page.Contents {
			contents = append(contents, o)
		}

		return true
	})

	fmt.Println(len(contents), " objects in the bucket.")

	// This is used to limit simultaneous goroutines
	throttle := make(chan int, attrs.Concurrency)
	var wg sync.WaitGroup

	// Loop trough the objects in the bucket and create a copy
	// of each object with the storage class, you've chosen
	bar := pb.StartNew(len(contents))
	for _, content := range contents {
		if *content.StorageClass != attrs.Type {
			throttle <- 1
			wg.Add(1)
			go func() {
				defer wg.Done()
				copyParams := &s3.CopyObjectInput{
					Bucket:       aws.String(attrs.Bucket),
					CopySource:   aws.String(attrs.Bucket + "/" + *content.Key),
					Key:          aws.String(*content.Key),
					StorageClass: aws.String(attrs.Type),
				}
				_, err := svc.CopyObject(copyParams)
				if err != nil {
					warns[*content.Key] = err
				}

				<-throttle
			}()
			wg.Wait()
		}
		bar.Increment()
	}
	bar.FinishPrint("Done!")
	// Fill the channel to be sure, that all goroutines finished
	for i := 0; i < cap(throttle); i++ {
		throttle <- 1
	}
	return warns
}

func main() {
	start := time.Now()
	var region, config string
	// Parsing arguments
	flag.Parse()
	if *bucketPtr == "" {
		color.Set(color.FgRed)
		fmt.Println("You haven't define bucket! Please, do it with -bucket= \n Script usage:")
		flag.PrintDefaults()
		log.Fatal("Bucket not specified")
		color.Unset()
		return
	}
	if *regionPtr == "" {
		region = "us-east-1"
		fmt.Println("You haven't specified region. Default region will be us-east-1")
	} else {
		region = *regionPtr
	}
	storageType := checkClass(*typePtr)

	attrs := attrs{
		Region:      region,
		Bucket:      *bucketPtr,
		Config:      config,
		Section:     *sectionPtr,
		Type:        storageType,
		Concurrency: *concurrencyPtr,
	}

	warns := convert(attrs)
	if len(warns) > 0 {
		logger(attrs.Bucket, warns)
	}
	elapsed := time.Since(start)
	log.Printf("Convertion took: %s", elapsed)
}
