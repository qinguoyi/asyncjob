package tools

import (
	"fmt"
	"github.com/qinguoyi/asyncjob/app/pkg/utils"
)

func GenerateBucketName(c chan<- string) {
	i := 0
	for {
		c <- fmt.Sprintf(utils.BucketName, i)
		i = (i + 1) % utils.BucketNum
	}
}
