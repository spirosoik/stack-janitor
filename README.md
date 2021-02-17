# stack-janitor

A lambda function which detects and remove cloudformation stacks periodically based
on the provided TAG and a specific expiration time in hours.

## Purpose

Respecting the environments to be clean and low cost we could run scheduled lambda function
which wipes out cloudformation stacks with specific tags.

## How to

We are going to schedule a CloudWatch event to invoke the lambda function periodically. Lambda function
will do the listing based on the provided tags and will wipe them out if they expired based on 
the max expiration hours we have set.

### Architecture

// TODO

## Deploy

```
make
```

The above will run the followings:
- Build Binary for lambda
- Prepare the artifact for Lambda
- Upload Lambda to provided S3 bucket
- Apply terraform (note: you need to provide env vars for terraform or use your own `.tfvars`)

For terraform details you can read the [documentation](terraform/aws/README.md)