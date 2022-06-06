package util

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type stackTracer interface {
	StackTrace() errors.StackTrace
}

// JoinStringMap generates a list of map element separated by string excluding keys in excluded maps
func JoinStringMap(m map[string]string, exclude map[string]string, sep string) []string {
	res := make([]string, len(m))
	i := 0
	for k, v := range m {
		res[i] = fmt.Sprintf("%s%s%s", k, sep, v)
		i++
	}
	return res
}

// StringSliceToSet converts a string slice to a set like map
func StringSliceToSet(lst []string) map[string]bool {
	res := make(map[string]bool)
	for _, el := range lst {
		res[el] = true
	}
	return res
}

// GetDestFileName builds the destination filename
func GetDestFileName(cacheDir string, cluster string, resourceName string) string {
	destDir := path.Join(cacheDir, cluster)
	destFileName := path.Join(destDir, resourceName)
	err := os.MkdirAll(destDir, os.ModePerm)
	FatalIf(err)
	return destFileName
}

// WriteStringToFile writes string to the given file and sync file
func WriteStringToFile(str string, destDir string, resourceName string, suffix string) error {
	name := fmt.Sprintf("%s_%s", resourceName, suffix)
	tempPattern := fmt.Sprintf("_%s_%s", resourceName, suffix)
	logrus.Infof("Writing file %s", name)
	tempFile, err := ioutil.TempFile(destDir, tempPattern)
	if err != nil {
		return errors.Wrapf(err, "Error creating temp file in %s for resource %s",
			destDir, name)
	}
	w := bufio.NewWriter(tempFile)
	_, err = w.WriteString(str)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return errors.Wrapf(err, "Error writing bytes to file %s",
			tempFile.Name())
	}
	err = w.Flush()
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return errors.Wrapf(err, "Error flushing buffer")
	}
	err = tempFile.Sync()
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return errors.Wrapf(err, "Error syncing file")
	}
	err = os.Rename(tempFile.Name(), path.Join(destDir, name))
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return err
	}
	return nil
}

// JoinSlicesOrNone joins a slice of string with separator or display None if there's no elements
func JoinSlicesOrNone(sl []string, sep string) string {
	if len(sl) == 0 {
		return "None"
	}
	return strings.Join(sl, sep)
}

// TruncateString truncates a string to a given maximum
func TruncateString(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

// JoinSlicesWithMaxOrNone joins a slice of string with separator up to x elements. Display None if there's no elements
func JoinSlicesWithMaxOrNone(sl []string, max int, sep string) string {
	if len(sl) == 0 {
		return "None"
	}
	if len(sl) < max {
		return strings.Join(sl, sep)
	}
	toDisplay := sl[:max]
	toDisplay = append(toDisplay, "...")
	return strings.Join(toDisplay, sep)
}

// StringMapsEqual returns true if maps are equals
func StringMapsEqual(a map[string]string, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if a[k] != b[k] {
			return false
		}
	}
	return true
}

// StringSlicesEqual returns true if slices are equals
func StringSlicesEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if a[k] != b[k] {
			return false
		}
	}
	return true
}

// DumpLine replaces empty string by None, join the slice and append newline
func DumpLine(lst []string) string {
	for k, v := range lst {
		if v == "" {
			lst[k] = "None"
		}
	}
	line := strings.Join(lst, " ")
	return fmt.Sprintf("%s\n", line)
}

// ExcludeFromSlice removes elements in exclude map from slice sl
func ExcludeFromSlice(sl []string, exclude map[string]string) []string {
	res := make([]string, len(sl))
	i := 0
	for k, v := range sl {
		_, isExcluded := exclude[v]
		if isExcluded {
			continue
		}
		res[k] = v
		i++
	}
	return res[:i]
}

// IsStringExcluded returns true if one of the regexp match the input string
func IsStringExcluded(s string, regexps []*regexp.Regexp) bool {
	for _, regexp := range regexps {
		if regexp.MatchString(s) {
			return true
		}
	}
	return false
}

// RegexpFilter removes elements in matching one of the regexp from slice sl
func FilterSliceWithRegexps(sl []string, excludeRegexps []*regexp.Regexp) []string {
	res := make([]string, len(sl))
	i := 0
	for k, s := range sl {
		if IsStringExcluded(s, excludeRegexps) {
			continue
		}
		res[k] = s
		i++
	}
	return res[:i]
}

// FatalIf exits if the error is not nil
func FatalIf(err error) {
	if err != nil {
		if stackErr, ok := err.(stackTracer); ok {
			logrus.WithField("stacktrace", fmt.Sprintf("%+v", stackErr.StackTrace()))
		} else {
			debug.PrintStack()
		}
		logrus.Fatalf("Fatal error: %s\n", err)
	}
}

// JoinIntSlice creates a string of joined int with a separator character
func JoinIntSlice(a []int, sep string) string {
	if len(a) == 0 {
		return "None"
	}
	b := make([]string, len(a))
	for i, v := range a {
		b[i] = strconv.Itoa(v)
	}
	return strings.Join(b, sep)
}

// LastURLPart extracts the last part of the url
func LastURLPart(url string) string {
	urlArray := strings.Split(url, "/")
	return urlArray[len(urlArray)-1]
}

// TimeToAge converts a time to a age string
func TimeToAge(t time.Time) string {
	duration := time.Now().Sub(t)
	duration = duration.Round(time.Minute)
	if duration.Hours() > 30 {
		return fmt.Sprintf("%dd", int(duration.Hours()/24))
	}
	hour := duration / time.Hour
	duration -= hour * time.Hour
	minute := duration / time.Minute
	return fmt.Sprintf("%02d:%02d", hour, minute)
}
