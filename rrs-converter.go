package main

import (
	"flag"
	"fmt"
	"log"
	"os/user"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"gopkg.in/cheggaaa/pb.v1"
)

// attrs store user-defined parameters
type attrs struct {
	Region      string
	Bucket      string
	Config      string
	Section     string
	Concurrency int
}

// Get user-defined parameters from CLI
var (
	bucketPtr      = flag.String("bucket", "", "Defines bucket. This is a mandatory paramenter!")
	regionPtr      = flag.String("region", "", "Defines region")
	configPtr      = flag.String("config", "", "Allow changing AWS account")
	sectionPtr     = flag.String("section", "default", "Which part of AWS credentials to use")
	concurrencyPtr = flag.Int("maxcon", 10, "Set up maximum concurrency for this task. Default is 10")
)

func convert(attrs attrs) {
	creds := credentials.NewSharedCredentials(attrs.Config, attrs.Section)
	_, err := creds.Get()
	if err != nil {
		log.Fatal(err)
		return
	}
	// Create new connection to S3
	svc := s3.New(session.New(), &aws.Config{
		Region:      aws.String(attrs.Region),
		Credentials: creds,
	})
	params := &s3.ListObjectsInput{
		Bucket: aws.String(attrs.Bucket),
	}
	resp, _ := svc.ListObjects(params)
	fmt.Println(len(resp.Contents), " objects in the bucket.")

	// This is used to limit simultaneous goroutines
	throttle := make(chan int, attrs.Concurrency)
	var wg sync.WaitGroup

	// Loop trough the objects in the bucket and create a copy
	// of each object with the REDUCED_REDUNDANCY storage class
	bar := pb.StartNew(len(resp.Contents))
	for _, content := range resp.Contents {
		if *content.StorageClass != "REDUCED_REDUNDANCY" {
			throttle <- 1
			wg.Add(1)
			go func() {
				defer wg.Done()
				copyParams := &s3.CopyObjectInput{
					Bucket:       aws.String(attrs.Bucket),
					CopySource:   aws.String(attrs.Bucket + "/" + *content.Key),
					Key:          aws.String(*content.Key),
					StorageClass: aws.String("REDUCED_REDUNDANCY"),
				}
				_, err := svc.CopyObject(copyParams)
				if err != nil {
					log.Fatal(err)
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
}

func main() {
	start := time.Now()
	var region, config string
	// Parsing arguments
	flag.Parse()
	if *bucketPtr == "" {
		fmt.Println("You haven't define bucket! Please, do it with -bucket= \n Script usage:")
		flag.PrintDefaults()
		log.Fatal("Bucket not specified")
		return
	}
	if *regionPtr == "" {
		region = "us-east-1"
		fmt.Println("You haven't specified region. Default region will be us-east-1")
	} else {
		region = *regionPtr
	}
	if *configPtr == "" {
		usr, err := user.Current()
		if err != nil {
			log.Fatal(err)
			return
		}
		config = usr.HomeDir + "/.aws/credentials"
	} else {
		config = *configPtr
	}
	attrs := attrs{
		Region:      region,
		Bucket:      *bucketPtr,
		Config:      config,
		Section:     *sectionPtr,
		Concurrency: *concurrencyPtr,
	}

	convert(attrs)
	elapsed := time.Since(start)
	log.Printf("Convertion took: %s", elapsed)
}
