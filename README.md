# ecr-scan-util
POC for aggregating/massaging output of ECR scans

```
usage: ecr-scan-util [<flags>] <command> [<args> ...]

(Common) Flags:
  --help                  Show context-sensitive help (also try --help-long and --help-man).
  --verbose               Log actions to stdout. Defaults to true.
  --registry-id=""        Aws ecr repository id. Uses default when omitted.
  --base-repo=""          Used when supplying image names with a common prefix
  --region="eu-west-1"    AWS region
  --latest-tag            Get result for most recent tagged image for specified repo. 
                          Ignores version of supplied composition if present.
  --latest-tag-filter=""  Ignores tags containing this substring.


Commands:
  help [<command>...]
    Show help.

  report [ all/single/composition ]

  common flags:
    --output-dir="reports"  Directory to write reports to
    --whitelist=""          Whitelist file containing package substrings to ignore per image and/or globally
    --cutoff="MEDIUM"       Severity cut off. Anything equal to or above is counted as a failures in the report
    --reporter="junit"      Reporter(s) to use, only JUnit for now.

  report all
    Iterate over all repositories in a given registry. (Finds latest tagged container and returns reports.)

  report single [<flags>]
    Iterate over a single repository

  flags:
    --image-id=""           Container name to fetch scan results for
    --image-tag=""          Container tag to fetch scan results for

  report composition [<flags>]
    Iterate over a user supplied list of Images (composition)
   
  flags:
    --compositionfile=""       ZD Composition file to load.
    --strip-prefix=""          Prefix string to strip while parsing composition entries. Removes first occurrence of substring.
    --strip-suffix="_version"  Suffix string to strip while pasrsing composition entries. Removes last occurrence of substring.

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
if you supply a baserepo containername is formatted like
`<baserepo>/<container>:<tag>`

if you do not it's formatted as 
`<container>:<tag>`

Note that aws registries usually have a prefix which you then need to include in the container 
### strip-prefix/suffix
Removes first or last occurrence of provided string from the container parameter, used to parse internal ZorgDomein composition files. 

### cutoff: 
findings have LOW, MEDIUM, HIGH, CRITICAL assesments. The JUnit reporter counts 'failures' by adding findings of cutoff.
or above. Case sensitive.

INFORMATIONAL is never counted. UNASSIGNED is counted as errors for the report as they require manual review.

### verbose: 
Boolean, whether to log to standard out. Defaults to true.

TODO/WANTS list:
Using hashes as identifier is on the TODO list, low priority.
