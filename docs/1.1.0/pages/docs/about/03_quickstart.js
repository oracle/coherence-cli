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
<p class="admonition-inline">See the <a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.0/rest-reference/quick-start.html">Coherence Documentation</a>
for more information on setting up Management over REST.</p>
</div>
<p>If you do not have a cluster running with Management over REST, you can run the following Docker
image to start a Coherence CE cluster and expose management on port 30000.</p>

<markup
lang="bash"

>docker run -d -p 30000:30000 ghcr.io/oracle/coherence-ce:21.12.1</markup>

</li>
</ol>
<p>If you are connecting to a Coherence Commercial version you must be on at least the following patch sets:</p>

<ul class="ulist">
<li>
<p>12.2.1.4.x - minimum patch level of 12.2.1.4.7+ required</p>

</li>
<li>
<p>14.1.1.0.x - minimum patch level of 14.1.1.0.2+ required</p>

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
CLI Version: 1.1.0
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
  clear       clear a context
  completion  generate the autocompletion script for the specified shell
  configure   configure tracing
  create      create a resource
  describe    show details of a specific resource
  discover    discover a cluster
  dump        dump a resource
  get         display one or many resources
  help        Help about any command
  log         log a resource
  nslookup    execute a Coherence Name Service lookup
  pause       pause a resource
  recover     recover a resource
  remove      remove a resource
  replicate   replicate a federated service
  resume      resume a resource
  retrieve    retrieve a resource
  set         set a configuration value
  shutdown    shutdown a resource
  start       start a resource
  stop        stop a resource
  suspend     suspend a resource
  version     show version information

Flags:
      --config string       config file (default is $HOME/.cohctl/cohctl.yaml)
      --config-dir string   config directory (default is $HOME/.cohctl)
  -c, --connection string   cluster connection name. (not required if context is set)
  -d, --delay int32         delay for watching in seconds (default 5)
  -h, --help                help for cohctl
  -o, --output string       output format: table, wide, json or jsonpath="..." (default "table")
  -i, --stdin               read password from stdin
  -U, --username string     basic auth username if authentication is required
  -w, --watch               watch output (only available for get commands)

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
CONNECTION  TYPE  URL                                                  VERSION  CLUSTER NAME  CLUSTER TYPE  CTX
local       http  http://localhost:30000/management/coherence/cluster  21.12    my-cluster    Standalone</markup>

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
Cluster Name:    cluster1
Version:         21.12
Cluster Size:    2
License Mode:    Development
Departure Count: 0
Running:         true

MACHINES
--------
MACHINE               PROCESSORS    LOAD  TOTAL MEMORY  FREE MEMORY  % FREE  OS     ARCH   VERSION
server1/192.168.16.3           4  0.0000       8.500GB      6.369GB  74.93%  Linux  amd64  5.10.47-linuxkit
/192.168.16.2                  4  0.1200       8.500GB      6.408GB  75.39%  Linux  amd64  5.10.47-linuxkit

MEMBERS
-------
Total cluster members: 2
Cluster Heap - Total: 4.254GB, Used: 161MB, Available: 4.097GB (96.3%)

SERVICES
--------
SERVICE NAME         TYPE              MEMBERS  STATUS HA     STORAGE  PARTITIONS
Proxy                Proxy                   2  n/a                -1          -1
PartitionedCache     DistributedCache        2  MACHINE-SAFE        2          31
ManagementHttpProxy  Proxy                   2  n/a                -1          -1

PERSISTENCE
-----------
Total Active Space Used: 0MB

SERVICE NAME       STORAGE COUNT  PERSISTENCE MODE  ACTIVE BYTES  ACTIVE SPACE  AVG LATENCY  MAX LATENCY  SNAPSHOTS  STATUS
PartitionedCache               2  active                  66,439           0MB      0.000ms          0ms          0  Idle

CACHES
------
Total Caches: 6, Total primary storage: 0MB

SERVICE            CACHE                 CACHE SIZE  BYTES   MB
PartitionedCache   customers                    123  4,016  0MB

TOPICS
------

PROXY SERVERS
-------------
NODE ID  HOST IP        SERVICE NAME  CONNECTIONS  BYTES SENT  BYTES REC
1        0.0.0.0:20000  Proxy                   0           0          0
2        0.0.0.0:20000  Proxy                   0           0          0

HTTP SERVERS
-------------
NODE ID  HOST IP        SERVICE NAME         SERVER TYPE                                    REQUESTS  ERRORS
1        0.0.0.0:30000  ManagementHttpProxy  com.tangosol.coherence.http.DefaultHttpServer       382       0
2        0.0.0.0:30000  ManagementHttpProxy  com.tangosol.coherence.http.DefaultHttpServer         0       0</markup>

<div class="admonition note">
<p class="admonition-inline">Depending upon the services and caches running in your cluster, you will see something slightly different.</p>
</div>
<div class="admonition note">
<p class="admonition-inline">You can also provide the <code>-v</code> (verbose) and <code>-o wide</code> (wide format) flags to display more details.</p>
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
Backup Count After Writebehind      :  [1]
Event Backlog                       :  0
Event Count                         :  0
...

Thread Count Min                    :  [1]
Thread Count Update Time            :  [1970-01-01T00:00:00.000+00:00]
Thread Idle Count                   :  2
Thread Pool Sizing Enabled          :  map[true:3]
Transport Backlogged Connection List:  [[] [] []]

SERVICE MEMBERS
---------------
NODE ID  THREADS  IDLE  THREAD UTIL  MIN THREADS    MAX THREADS
      1        1     1        0.00%            2  2,147,483,647
      2        1     1        0.00%            2  2,147,483,647


SERVICE CACHES
--------------
Total Caches: 2, Total primary storage: 30MB

SERVICE           CACHE  CACHE SIZE       BYTES    MB
PartitionedCache  test1     100,000  26,288,448  25MB
PartitionedCache  test2      23,000   5,999,040   5MB

PERSISTENCE FOR SERVICE
-----------------------
Total Active Space Used: 0MB

NODE ID  PERSISTENCE MODE  ACTIVE BYTES  ACTIVE SPACE  AVG LATENCY  MAX LATENCY
      1  active                  44,560           0MB      0.000ms          0ms
      2  active                  21,879           0MB      0.000ms          0ms

PERSISTENCE COORDINATOR
-----------------------
Coordinator Id  :  1
Idle            :  true
Operation Status:  Idle
Snapshots       :  []</markup>

<div class="admonition note">
<p class="admonition-inline">The output above has been truncated for brevity.</p>
</div>
</div>

<h3 id="step7">7. Watching data</h3>
<div class="section">
<p>For all the <code>get</code> commands, you can add the <code>-w</code> option to watch the resource continuously until <code>CTRL-C</code>
has been pressed.  In the example below we are watching the cluster members.</p>

<markup
lang="bash"

>$ cohctl get members -w
2021-10-13 15:09:52.027636 +0800 AWST m=+0.059034774
Using cluster connection 'local' from current context.

Cluster Heap - Total: 1.750GB, Used: 330MB, Available: 1.428GB (81.6%)

NODE ID  ADDRESS          PORT  PROCESS  MEMBER  ROLE              MAX HEAP  USED HEAP  AVAIL HEAP
1        /192.168.1.121  57133  13919    n/a     Management           512MB      186MB       326MB
2        /192.168.1.121  57136  13941    n/a     CoherenceServer    1.000GB       67MB       957MB
3        /192.168.1.121  57169  13981    n/a     CoherenceConsole     256MB       77MB       179MB

2021-10-13 15:09:57.045536 +0800 AWST m=+5.076898506
Using cluster connection 'local' from current context.

Cluster Heap - Total: 1.750GB, Used: 332MB, Available: 1.426GB (81.5%)

NODE ID  ADDRESS          PORT  PROCESS  MEMBER  ROLE              MAX HEAP  USED HEAP  AVAIL HEAP
1        /192.168.1.121  57133  13919    n/a     Management           512MB      186MB       326MB
2        /192.168.1.121  57136  13941    n/a     CoherenceServer    1.000GB       68MB       956MB
3        /192.168.1.121  57169  13981    n/a     CoherenceConsole     256MB       77MB       179MB</markup>

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
