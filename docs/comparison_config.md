# Comparison Config

## BaseBuild 

*BuildConfig*

### Label

*string*

A label identifying the build.

### URL

*string*

URL from where to download a build release. This can also point to a local file if prefixed with "file://". In such case, the build file will be uploaded to the app servers.

## NewBuild

*BuildConfig*

### Label

*string*

A label identifying the build.

### URL

*string*

URL from where to download a build release. This can also point to a local file if prefixed with "file://". In such case, the build file will be uploaded to the app servers.

## LoadTests 

*[]LoadTestConfig*

### Type 

*LoadTestType*

The type of load-test to run.

Possible values:
- "bounded"
- "unbounded"

### DBEngine

*DatabaseEngine*

The database engine for the app server.

Possible values:
- "mysql"
- "postgresql"

### NumUsers

*int*

The number of users to run. This is only considered if Type is "bounded"

### Duration 

*string*

The duration of the load-test. This is only considered if Type is "bounded"

## Output

*OutputConfig*

### UploadDashboard 

*bool*

A boolean indicating whether a comparative Grafana dashboard should be generated and uploaded.

### GenerateReport

*bool*

A boolean indicating whether to generate a markdown report at the end of the comparison.

### GenerateGraphs

*bool*

A boolean indicating whether to generate gnuplot graphs at the end of the comparison.

### GraphsPath 

*string*

An optional path indicating where to write the graphs.
