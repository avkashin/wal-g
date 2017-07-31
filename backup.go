package walg

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"io"
	"sort"
	"strings"
)

type WalFiles interface {
	CheckExistence() bool
}

type ReaderMaker interface {
	Reader() io.ReadCloser
	Format() string
	Path() string
}

type S3ReaderMaker struct {
	Backup     *Backup
	Key        *string
	FileFormat string
}

func (s *S3ReaderMaker) Format() string { return s.FileFormat }
func (s *S3ReaderMaker) Path() string   { return *s.Key }

/**
 *  Create a new S3 reader for each object.
 */
func (s *S3ReaderMaker) Reader() io.ReadCloser {
	input := &s3.GetObjectInput{
		Bucket: s.Backup.Prefix.Bucket,
		Key:    s.Key,
	}

	rdr, err := s.Backup.Prefix.Svc.GetObject(input)
	if err != nil {
		panic(err)
	}
	return rdr.Body

}

type Prefix struct {
	Svc    s3iface.S3API
	Bucket *string
	Server *string
}

type Backup struct {
	Prefix *Prefix
	Path   *string
	Name   *string
	Js     *string
}

/**
 *  Sorts the backups by last modified time and returns the latest backup key.
 */
func (b *Backup) GetLatest() string {
	objects := &s3.ListObjectsV2Input{
		Bucket:    b.Prefix.Bucket,
		Prefix:    b.Path,
		Delimiter: aws.String("/"),
	}

	backups, err := b.Prefix.Svc.ListObjectsV2(objects)
	if err != nil {
		panic(err)
	}

	sortTimes := make([]BackupTime, len(backups.Contents))

	for i, ob := range backups.Contents {
		key := *ob.Key
		time := *ob.LastModified
		sortTimes[i] = BackupTime{stripNameBackup(key), time}
	}

	sort.Sort(TimeSlice(sortTimes))

	return sortTimes[0].Name
}

/**
 *  Strips the backup key and returns it in its base form `base_...`.
 */
func stripNameBackup(key string) string {
	all := strings.SplitAfter(key, "/")
	name := strings.Split(all[2], "_backup")[0]
	return name
}

/**
 *  Checks that the specified backup exists.
 */
func (b *Backup) CheckExistence() bool {
	js := &s3.HeadObjectInput{
		Bucket: b.Prefix.Bucket,
		Key:    b.Js,
	}

	_, err := b.Prefix.Svc.HeadObject(js)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "NotFound":
				return false
			}
		}
	}
	return true
}

/**
 *  Gets the keys of the files in the specified backup.
 */
func (b *Backup) GetKeys() []string {
	objects := &s3.ListObjectsV2Input{
		Bucket: b.Prefix.Bucket,
		Prefix: aws.String(*b.Path + *b.Name + "/tar_partitions"),
	}

	files, err := b.Prefix.Svc.ListObjectsV2(objects)
	if err != nil {
		panic(err)
	}

	arr := make([]string, len(files.Contents))

	for i, ob := range files.Contents {
		key := *ob.Key
		arr[i] = key
	}

	return arr
}

type Archive struct {
	Prefix  *Prefix
	Archive *string
}

/**
 *  Checks that the specified WAL file exists.
 */
func (a *Archive) CheckExistence() bool {
	arch := &s3.HeadObjectInput{
		Bucket: a.Prefix.Bucket,
		Key:    a.Archive,
	}

	_, err := a.Prefix.Svc.HeadObject(arch)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "NotFound":
				return false
			}
		}
	}
	return true
}

/**
 *  Downloads the specified archive from S3.
 */
func (a *Archive) GetArchive() io.ReadCloser {
	input := &s3.GetObjectInput{
		Bucket: a.Prefix.Bucket,
		Key:    a.Archive,
	}

	archive, err := a.Prefix.Svc.GetObject(input)
	if err != nil {
		panic(err)
	}

	return archive.Body
}
