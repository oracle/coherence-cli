<doc-view>

<h2 id="_overview">Overview</h2>
<div class="section">
<p>This guide is a simple set of steps to get you started with the Coherence CLI.</p>

</div>

<h2 id="_prerequisites">Prerequisites</h2>
<div class="section">
<ol style="margin-left: 15px;">
<li>
You must have downloaded and installed the CLI for your platform as described in the
<router-link to="/docs/installation/01_installation">Coherence CLI Installation section</router-link>.

</li>
<li>
You must have a Coherence cluster running that has Management over REST configured.
<div class="admonition note">
<p class="admonition-inline">See the <a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/rest-reference/quick-start.html">Coherence Documentation</a>
for more information on setting up Management over REST.</p>
</div>
<p>If you do not have a cluster running with Management over REST, you can run the following Docker
image to start a Coherence CE cluster and expose management on port 30000.</p>

<markup
lang="bash"

>docker run -d -p 30000:30000 ghcr.io/oracle/coherence-ce:22.06.5</markup>

</li>
</ol>
<p>If you are connecting to a Coherence Commercial version you must be on at least the following patch sets:</p>

<ul class="ulist">
<li>
<p>12.2.1.4.x - minimum patch level of 12.2.1.4.10+ required</p>

</li>
<li>
<p>14.1.1.0.x - minimum patch level of 14.1.1.0.5+ required</p>

</li>
</ul>
<div class="admonition note">
<p class="admonition-inline">If you are connecting to WebLogic Server then see the <router-link to="/docs/reference/05_clusters">Clusters Command Reference</router-link> for more information on the connection requirements.</p>
</div>
</div>

<h2 id="_contents">Contents</h2>
<div class="section">
<ol style="margin-left: 15px;">
<li>
<router-link to="#step1" @click.native="this.scrollFix('#step1')">Display the Coherence CLI version</router-link>

</li>
<li>
<router-link to="#step2" @click.native="this.scrollFix('#step2')">Display CLI usage</router-link>

</li>
<li>
<router-link to="#step3" @click.native="this.scrollFix('#step3')">Add a cluster connection</router-link>

</li>
<li>
<router-link to="#step4" @click.native="this.scrollFix('#step4')">Describe the cluster</router-link>

</li>
<li>
<router-link to="#step5" @click.native="this.scrollFix('#step5')">Display services</router-link>

</li>
<li>
<router-link to="#step6" @click.native="this.scrollFix('#step6')">Describe a service</router-link>

</li>
<li>
<router-link to="#step7" @click.native="this.scrollFix('#step7')">Watching data</router-link>

</li>
<li>
<router-link to="#step8" @click.native="this.scrollFix('#step8')">Change the output format to Json and using JsonPath</router-link>

</li>
</ol>

<h3 id="step1">1. Display the Coherence CLI version</h3>
<div class="section">
<p>Issue the following command to show the version details of the tool you are using.</p>

<markup
lang="bash"

>$ cohctl version

Coherence Command Line Interface
CLI Version: 1.5.1
Date:        2021-10-13T02:36:48Z
Commit:      6d1266bb473dad224a3672367126381263af
OS:          darwin
OS Version:  amd64</markup>

<div class="admonition note">
<p class="admonition-inline">THE CLI creates a hidden directory off the users home directory called <code>.cohctl</code> to store the cluster connection information plus other
information. You can issue <code>ls -l ~/.cohctl</code> on Mac/Linux to view the directory contents.</p>
</div>
</div>

<h3 id="step2">2. Display CLI usage</h3>
<div class="section">
<p>If you run <code>cohctl</code> without any arguments you will see the usage.
These options are explained in detail in  <router-link to="/docs/config/05_global_flags">Global Flags</router-link>.</p>

<markup
lang="bash"

>$ cohctl --help

The Coherence Command Line Interface (CLI) provides a way to
interact with, and monitor Coherence clusters via a terminal-based interface.

Usage:
  cohctl [command]

Available Commands:
  add         add a resource
  archive     archive a resource
  clear       clears resources
  compact     compact an elastic-data resource
  completion  Generate the autocompletion script for the specified shell
  configure   configure tracing
  connect     connect a resource
  create      create a resource
  describe    show details of a specific resource
  disconnect  disconnect a resource
  discover    discover a cluster
  dump        dump a resource
  get         display one or many resources
  help        Help about any command
  log         log a resource
  notify      notify a resource
  nslookup    execute a Coherence Name Service lookup
  pause       pause a resource
  recover     recover a resource
  remove      remove a resource
  replicate   replicate a federated service
  reset       reset statistics for various resources
  resume      resume a resource
  retrieve    retrieve a resource
  scale       scale a cluster
  set         set a configuration value
  shutdown    shutdown a resource
  start       start a resource
  stop        stop a resource
  suspend     suspend a resource
  truncate    truncates resources
  version     show version information

Flags:
  -b, --bytes               show sizes in bytes
      --config string       config file (default is $HOME/.cohctl/cohctl.yaml)
      --config-dir string   config directory (default is $HOME/.cohctl)
  -c, --connection string   cluster connection name. (not required if context is set)
  -d, --delay int32         delay for watching in seconds (default 5)
  -g, --gb                  show sizes in gigabytes (default is bytes)
  -h, --help                help for cohctl
  -k, --kb                  show sizes in kilobytes (default is bytes)
  -m, --mb                  show sizes in megabytes (default is bytes)
  -o, --output string       output format: table, wide, json or jsonpath="..." (default "table")
  -i, --stdin               read password from stdin
      --tb                  show sizes in terabytes (default is bytes)
  -U, --username string     basic auth username if authentication is required
  -w, --watch               watch output (only available for get commands)
  -W, --watch-clear         watch output with clear

Use "cohctl [command] --help" for more information about a command.</markup>

</div>

<h3 id="step3">3. Add a cluster connection</h3>
<div class="section">
<p>Next, you must add a connection to a Coherence cluster.  In this example the cluster is running on
the local machine and the Management over REST port is 30000. Adjust for your Coherence cluster.</p>

<p>When you add a cluster connection you give it a context name, which may be different that the cluster name and url to
connect to.</p>

<markup
lang="bash"

>$ cohctl add cluster local -u http://localhost:30000/management/coherence/cluster
Added cluster local with type http and URL http://localhost:30000/management/coherence/cluster

$ cohctl get clusters
CONNECTION  TYPE  URL                                                  VERSION  CLUSTER NAME  CLUSTER TYPE  CTX  LOCAL
local       http  http://localhost:30000/management/coherence/cluster  21.12    my-cluster    Standalone         false</markup>

<div class="admonition note">
<p class="admonition-inline">If you are not using a Docker container, you can also use the <code>cohctl discover clusters</code> command to automatically discover clusters using the Name Service.</p>
</div>
</div>

<h3 id="step4">4. Describe the cluster</h3>
<div class="section">
<p>Now that the cluster connection is added, you can describe the cluster using the <code>describe cluster</code> command.</p>

<markup
lang="bash"

>$ cohctl describe cluster local
CLUSTER
-------
Cluster Name:    my-cluster
Version:         21.12.4
Cluster Size:    3
License Mode:    Development
Departure Count: 0
Running:         true

MACHINES
--------
MACHINE        PROCESSORS    LOAD    TOTAL MEMORY    FREE MEMORY  % FREE  OS        ARCH    VERSION
192.168.1.117           8  3.9067  34,359,738,368  1,671,991,296   4.87%  Mac OS X  x86_64  10.16

MEMBERS
-------
Total cluster members: 3
Cluster Heap - Total: 4,563,402,752 Used: 171,966,464 Available: 4,391,436,288 (96.2%)
Storage Heap - Total: 4,294,967,296 Used: 117,440,512 Available: 4,177,526,784 (97.3%)

SERVICES
--------
SERVICE NAME         TYPE              MEMBERS  STATUS HA   STORAGE  PARTITIONS
Proxy                Proxy                   2  n/a              -1          -1
PartitionedTopic     DistributedCache        2  NODE-SAFE         2         257
PartitionedCache2    DistributedCache        2  NODE-SAFE         2         257
PartitionedCache     DistributedCache        3  NODE-SAFE         2         257
ManagementHttpProxy  Proxy                   1  n/a              -1          -1
"$SYS:Config"        DistributedCache        2  ENDANGERED        2         257

PERSISTENCE
-----------
Total Active Space Used: 0

SERVICE NAME       STORAGE COUNT  PERSISTENCE MODE  ACTIVE SPACE  AVG LATENCY  MAX LATENCY  SNAPSHOTS  STATUS
PartitionedTopic               2  on-demand                    0      0.000ms          0ms          1  Idle
PartitionedCache2              2  on-demand                    0      0.000ms          0ms          3  Idle
PartitionedCache               2  on-demand                    0      0.000ms          0ms          1  Idle
"$SYS:Config"                  2  on-demand                    0      0.000ms          0ms          0  Idle

CACHES
------
Total Caches: 2, Total primary storage: 19,030,328

SERVICE           CACHE       COUNT        SIZE
PartitionedCache  customers  50,230  13,204,808
PartitionedCache  orders      5,022   5,825,520

TOPICS
------

PROXY SERVERS
-------------
NODE ID  HOST IP              SERVICE NAME  CONNECTIONS  BYTES SENT  BYTES REC
1        0.0.0.0:63984.55876  Proxy                   0           0          0
2        0.0.0.0:63995.53744  Proxy                   0           0          0

HTTP SERVERS
-------------
NODE ID  HOST IP        SERVICE NAME         SERVER TYPE                                 REQUESTS  ERRORS
1        0.0.0.0:30000  ManagementHttpProxy  com.tangosol.coherence.http.JavaHttpServer         0       0</markup>


<h4 id="_notes">Notes</h4>
<div class="section">
<ol style="margin-left: 15px;">
<li>
Depending upon the services and caches running in your cluster, you will see something slightly different.

</li>
<li>
You can also provide the <code>-v</code> (verbose) and <code>-o wide</code> (wide format) flags to display more details.

</li>
<li>
By default, all memory and disk values are displayed in bytes as you can see above.
You can change this by specifying <code>-k</code> for KB, <code>-m</code> for MB or <code>-g</code> for GB. This applies to all memory or disk values returned.

</li>
</ol>
</div>
</div>

<h3 id="step5">5. Display services</h3>
<div class="section">
<p>You can issue various <code>get</code> commands to display different resources. Issue the <code>get services</code> command to
show the services for the cluster only.</p>

<markup
lang="bash"

>$ cohctl get services -c local

SERVICE NAME         TYPE              MEMBERS  STATUS HA     STORAGE  PARTITIONS
Proxy                Proxy                   2  n/a                -1          -1
PartitionedCache     DistributedCache        2  MACHINE-SAFE        2          31
ManagementHttpProxy  Proxy                   2  n/a                -1          -1</markup>

<p>All commands other than <code>describe cluster</code> require a <code>-c</code> option to specify the cluster you wish to
connect to. You can use the <code>cohctl set context &lt;name&gt;</code> to specify the context (or cluster
connection) you are working  with, so you don&#8217;t have to specify <code>-c</code> each time.</p>

<markup
lang="bash"

>$ cohctl set context local
Current context is now local

$ cohctl get services
Using cluster connection 'local' from current context.

SERVICE NAME         TYPE              MEMBERS  STATUS HA     STORAGE  PARTITIONS
Proxy                Proxy                   2  n/a                -1          -1
PartitionedCache     DistributedCache        2  MACHINE-SAFE        2          31
ManagementHttpProxy  Proxy                   2  n/a                -1          -1</markup>

</div>

<h3 id="step6">6. Describe a service</h3>
<div class="section">
<p>Above we have issued a <code>get services</code> command and for all resources you can use a <code>describe</code> command to
show specific details about a resource, or a service in our case.</p>

<p>The output from a <code>describe</code> command will usually contain much more detailed information about the resource.</p>

<markup
lang="bash"

>$ cohctl describe service PartitionedCache

Using cluster connection 'local' from current context.

SERVICE DETAILS
---------------
Name                                :  PartitionedCache
Type                                :  [DistributedCache]
Backup Count                        :  [1]
...

SERVICE MEMBERS
---------------
NODE ID  THREADS  IDLE  THREAD UTIL  MIN THREADS    MAX THREADS
      1        1     1        0.00%            1  2,147,483,647
      2        1     1        0.00%            1  2,147,483,647
      3        0    -1          n/a            1  2,147,483,647

SERVICE CACHES
--------------
Total Caches: 2, Total primary storage: 37,888,448

SERVICE           CACHE        COUNT        SIZE
PartitionedCache  customers  100,000  26,288,448
PartitionedCache  orders      10,000  11,600,000

PERSISTENCE FOR SERVICE
-----------------------
Total Active Space Used: 0

NODE ID  PERSISTENCE MODE  ACTIVE SPACE  AVG LATENCY  MAX LATENCY
      1  on-demand                    0      0.000ms          0ms
      2  on-demand                    0      0.000ms          0ms

PERSISTENCE COORDINATOR
-----------------------
Coordinator Id  :  1
Idle            :  true
Operation Status:  Idle
Snapshots       :  [snapshot-1]

DISTRIBUTION INFORMATION
------------------------
Scheduled Distributions:  No distributions are currently scheduled for this service.

PARTITION INFORMATION
---------------------
Service                     :  PartitionedCache
Strategy Name               :  SimpleAssignmentStrategy
Average Partition Size KB   :  143
Average Storage Size KB     :  18500
Backup Count                :  1
...
Service Node Count          :  2
Service Rack Count          :  1
Service Site Count          :  1
Type                        :  PartitionAssignment</markup>

<div class="admonition note">
<p class="admonition-inline">The output above has been truncated for brevity.</p>
</div>
</div>

<h3 id="step7">7. Watching data</h3>
<div class="section">
<p>For all the <code>get</code> commands, you can add the <code>-w</code> option to watch the resource continuously until <code>CTRL-C</code>
has been pressed.  In the example below we are watching the cluster members.</p>

<div class="admonition note">
<p class="admonition-inline">We are setting the <code>-m</code> option to show sizes in MB rather than the default of bytes.</p>
</div>
<markup
lang="bash"

>$ cohctl get members -w -m

2022-04-27 15:11:24.393725 +0800 AWST m=+0.031247503
Using cluster connection 'local' from current context.

Total cluster members: 3
Cluster Heap - Total: 4,352 MB Used: 261 MB Available: 4,091 MB (94.0%)
Storage Heap - Total: 4,096 MB Used: 212 MB Available: 3,884 MB (94.8%)

NODE ID  ADDRESS        PORT   PROCESS  MEMBER  ROLE                  STORAGE  MAX HEAP  USED HEAP  AVAIL HEAP
      1  192.168.1.117  63984    35372  n/a     Management            true     2,048 MB     127 MB    1,921 MB
      2  192.168.1.117  63995    35398  n/a     TangosolNetCoherence  true     2,048 MB      85 MB    1,963 MB
      3  192.168.1.117  64013    35430  n/a     CoherenceConsole      false      256 MB      49 MB      207 MB

2022-04-27 15:11:29.419558 +0800 AWST m=+5.057038216
Using cluster connection 'local' from current context.

Total cluster members: 3
Cluster Heap - Total: 4,352 MB Used: 263 MB Available: 4,089 MB (94.0%)
Storage Heap - Total: 4,096 MB Used: 214 MB Available: 3,882 MB (94.8%)

NODE ID  ADDRESS         PORT   PROCESS  MEMBER  ROLE                  STORAGE  MAX HEAP  USED HEAP  AVAIL HEAP
      1  192.168.1.117  63984    35372  n/a     Management            true     2,048 MB     129 MB    1,919 MB
      2  192.168.1.117  63995    35398  n/a     TangosolNetCoherence  true     2,048 MB      85 MB    1,963 MB
      3  192.168.1.117  64013    35430  n/a     CoherenceConsole      false      256 MB      49 MB      207 MB</markup>

<div class="admonition note">
<p class="admonition-inline">You can change the delay from the default of 5 seconds by using <code>-d</code> option and specifying the seconds
to delay, e.g. <code>cohctl get members -w -d 10</code>.</p>
</div>
<div class="admonition note">
<p class="admonition-inline">You can also use <code>-o wide</code> to display more columns on most commands.</p>
</div>
</div>

<h3 id="step8">8. Change the output format to Json and using JSONPath</h3>
<div class="section">
<p>The default output format is text, but you can specify <code>-o json</code> on any command to get the output
in Json format. You can also use <code>-o jsonpath="&#8230;&#8203;"</code> to apply a JsonPath expression.</p>

<p>Below we are changing the format for the <code>get members</code> to be Json and piping it thought the <code>jq</code> utility to format.</p>

<markup
lang="bash"

>$ cohctl get members -o json | jq

{
  "items": [
    {
      "processName": "13981",
      "socketCount": -1,
      "siteName": "n/a",
      "publisherSuccessRate": 1,
      "trafficJamCount": 8192,
      "multicastEnabled": true,
      "refreshTime": "2021-10-13T15:12:58.476+08:00",
...
}</markup>

<p>We can also JSONPath expressions to select or
query json output from any command. In the example below we get all service members
where the requestAverageDuration &gt; 15 millis.</p>

<markup
lang="bash"

>$ cohctl get services -o jsonpath="$.items[?(@.requestAverageDuration &gt; 15)]..['nodeId','name','requestAverageDuration']"
[
  "2",
  "PartitionedCache2",
  25.51414,
  "1",
  "PartitionedCache2",
  19.662437
]</markup>

<p>See the <router-link to="/docs/examples/10_jsonpath">JSONPath</router-link> examples for more information.</p>

</div>

<h3 id="_next_steps">Next Steps</h3>
<div class="section">
<p>Explore more of the commands <router-link to="/docs/reference/01_overview">here</router-link>.</p>

</div>
</div>
</doc-view>
