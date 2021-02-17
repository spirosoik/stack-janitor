## Providers

| Name | Version |
|------|---------|
| aws | n/a |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:-----:|
| environment | The environment where the lambda function runs | `string` | n/a | yes |
| filter\_tag\_key | The key name of the tag you want to add as a rule to be checked | `string` | n/a | yes |
| filter\_tag\_value | The value name of the tag you want to add as a rule to be checked for the provided key | `string` | n/a | yes |
| function\_name | The name of the lambda function | `string` | `"credentials-janitor"` | no |
| janitor\_lambda\_schedule | The schedule of triggering a cloudwatch event to invoke lambda | `string` | `"cron(0 * * * *)"` | no |
| lambda\_timeout | The name of the lambda function | `number` | `120` | no |
| max\_expiration\_hours | The number of hours for the rule about revoking credentials | `number` | `1` | no |
| private\_subet\_ids | n/a | `list(string)` | n/a | yes |
| s3\_bucket | The name of the bucket where the lambda artifact will exist | `string` | n/a | yes |
| s3\_key | The key/path where the lambda artifact will exist | `string` | n/a | yes |

## Outputs

No output.

