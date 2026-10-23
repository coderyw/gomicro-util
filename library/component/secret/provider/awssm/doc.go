// Package awssm AWS Secrets Manager Provider（ok/nwr 线使用）。
//
// 凭证走 aws-sdk-go-v2 默认链（IRSA 自动生效，与 object-storage/provider/s3 同一模式）；
// region 取 AWS_REGION / AWS_DEFAULT_REGION，缺省 ap-southeast-1。
package awssm
