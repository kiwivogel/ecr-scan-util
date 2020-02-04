# ecr-scan-util
POC for aggregating/massaging output of ECR scans
```
Flags:
  --help                     Show context-sensitive help (also try --help-long and --help-man).
  --repository=""            Aws ecr repository id. Uses default when omitted.
  --composition=             ZD Composition file to load when running batch mode.
  --strip-prefix=""          Prefix string to strip while composition entries. Removes first occurrence of substring.
  --strip-suffix="_version"  Suffix string to strip while composition entries. Removes last occurrence of substring.
  --baserepo=""              Prefix for images. will be prefixed onto entries in composition or containername supplied .
  --container=""             Container name to fetch scan results for
  --tag=""                   Container tag to fetch scan results for
  --directory="reports"      Directory to write reports to
  --cutoff="MEDIUM"          Severity to count as failures
  --verbose                  log actions to stdout
  --reporter="junit"         Reporter(s) to use

```

### composition: reads a yaml file with format: 
```yaml
postgresql_version: 'TAG'
yourcontainer: 'TAG'
zd_somecontainer_version: 'TAG'
```

NB. the parser will strip zd_ and _version and bump underscores to hyphens.

## Container format
### repository: repository name of ECR repository
ECR repository, defaults to URI for account associated with supplied credentials, which ok for most usecases

### baserepo / container / tag  
`<baserepo>/<container>:<tag>`

### cutoff: 
findings have LOW, MEDIUM, HIGH, CRITICAL assesments. The JUnit reporter counts 'failures' by adding findings of cutoff.
or above. Case sensitive.

INFORMATIONAL is never counted. UNASSIGNED is counted as errors for the report as they require manual review.

### verbose: 
Boolean, whether to log to standard out. Defaults to true for now.

TODO/WANTS list:
Using hashes as identifier is on the TODO list, low priority.
Program also currently assumes there's test results for the provided tags, add logic to handle this.
