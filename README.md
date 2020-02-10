# ecr-scan-util
POC for aggregating/massaging output of ECR scans
```
Flags:
  --help                     Show context-sensitive help (also try --help-long and --help-man).
  --repository=""            Aws ecr repository id. Uses default when omitted.
  --composition=""           ZD Composition file to load when running batch mode.
  --whitelist=""             Whitelist file containing package substrings to ignore per image and/or globally
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

### whitelist 
Allows passing a whitelist with packages that you want to allow in your scan results. Mainly used because Claire includes 
dummy kernel packages in results. Whitelisted packages can be supplied globally or on a per container basis in te following 
format.

N.B (Whitelist entries are parsed with 'HasPrefix' for now. More elaborate logic could be added)
```yaml
container_whitelist:
  jenkins:
    - systemd@232-25+deb9u12
  somecontainer:
    - somepackage
    - someotherpackage@someversion
  redis:
    - busybox
global_whitelist:
  - linux@4.9
```

## Container format
### repository: repository name of ECR repository
ECR repository, defaults to URI for account associated with supplied credentials, which ok for most usecases

### baserepo / container / tag  
`<baserepo>/<container>:<tag>`

### strip-prefix/suffix
Removes first or last occurrence of provided string from the container parameter, used to parse internal ZD composition files. 

### cutoff: 
findings have LOW, MEDIUM, HIGH, CRITICAL assesments. The JUnit reporter counts 'failures' by adding findings of cutoff.
or above. Case sensitive.

INFORMATIONAL is never counted. UNASSIGNED is counted as errors for the report as they require manual review.

### verbose: 
Boolean, whether to log to standard out. Defaults to true for now.

TODO/WANTS list:
Using hashes as identifier is on the TODO list, low priority.
