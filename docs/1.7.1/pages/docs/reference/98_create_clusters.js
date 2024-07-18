<doc-view>

<h2 id="_creating_clusters">Creating Clusters</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>There are various cluster commands that allow you to create and work with local development based clusters.</p>

<p>These commands allow you to create local only development based clusters (scoped to localhost) for Coherence CE 22.06.x and
Commercial 14.1.1.2206.x versions and above. You must have the following setup for this to work:</p>

<ol style="margin-left: 15px;">
<li>
Java 17+ executable on the PATH, if you are using the default Coherence CE 22.06.9 version.

</li>
<li>
Maven 3.6+ or JDK compatible Gradle version 7.+ executable on the PATH.

</li>
</ol>
<div class="admonition note">
<p class="admonition-inline">Maven is used by default for dependency resolution and classpath generation, but if you prefer Gradle, then use the
following: <code>cohctl set use-grade true</code>. You can revert to Maven by using <code>cohctl set use-gradle false</code>.</p>
</div>
<p>Once you create a local cluster, you can manage and monitor these clusters in the same way as you can with discovered
or manually added clusters. You can also run the Coherence console and CohQL (Query client) against these clusters.</p>

<p>When you create a cluster, the first node started will always have management over REST enabled.</p>

<p>You may also specify a profile when you start a cluster using the <code>-P</code> option. The value of the profile
, which is a string containing <code>-Dkey=value</code> pairs, will be included in the cache servers started.</p>

<p>On first creation of a Coherence cluster, if your Maven or Gradle repository is empty, it may take a short while to download the minimal depdencies.</p>

<div class="admonition note">
<p class="admonition-inline">These commands are experimental only and may be changed or removed in the future. It is <strong>not supported</strong> to use
these commands to create production clusters.</p>
</div>
<p><strong>Creating and controlling clusters</strong></p>

<ul class="ulist">
<li>
<p><router-link to="#create-cluster" @click.native="this.scrollFix('#create-cluster')"><code>cohctl create cluster</code></router-link> - creates a local cluster and adds to the cohctl.yaml file</p>

</li>
<li>
<p><router-link to="#scale-cluster" @click.native="this.scrollFix('#scale-cluster')"><code>cohctl scale cluster</code></router-link> - scales a cluster that was manually created</p>

</li>
<li>
<p><router-link to="#stop-cluster" @click.native="this.scrollFix('#stop-cluster')"><code>cohctl stop cluster</code></router-link> - stops a cluster that was manually created or started</p>

</li>
<li>
<p><router-link to="#start-cluster" @click.native="this.scrollFix('#start-cluster')"><code>cohctl start cluster</code></router-link> - starts a cluster that was manually created</p>

</li>
<li>
<p><router-link to="#start-console" @click.native="this.scrollFix('#start-console')"><code>cohctl start console</code></router-link> - starts a console client against a cluster that was manually created</p>

</li>
<li>
<p><router-link to="#start-cohql" @click.native="this.scrollFix('#start-cohql')"><code>cohctl start cohql</code></router-link> - starts a CohQL client against a cluster that was manually created</p>

</li>
<li>
<p><router-link to="#start-class" @click.native="this.scrollFix('#start-class')"><code>cohctl start class</code></router-link> - starts a specific Java class against a cluster that was manually created</p>

</li>
</ul>
<p><strong>Setting dependency tool</strong></p>

<ul class="ulist">
<li>
<p><router-link to="#set-use-gradle" @click.native="this.scrollFix('#set-use-gradle')"><code>cohctl set use-gradle</code></router-link> - sets whether to use gradle for dependency management</p>

</li>
<li>
<p><router-link to="#get-use-gradle" @click.native="this.scrollFix('#get-use-gradle')"><code>cohctl get use-gradle</code></router-link> - displays the current setting for using gradle for dependency management</p>

</li>
</ul>
<p><strong>Setting default heap sizes</strong></p>

<ul class="ulist">
<li>
<p><router-link to="#set-default-heap" @click.native="this.scrollFix('#set-default-heap')"><code>cohctl set default-heap</code></router-link> - sets default heap for creating and starting clusters</p>

</li>
<li>
<p><router-link to="#get-default-heap" @click.native="this.scrollFix('#get-default-heap')"><code>cohctl get default-heap</code></router-link> - gets default heap for creating and starting clusters</p>

</li>
<li>
<p><router-link to="#clear-default-heap" @click.native="this.scrollFix('#clear-default-heap')"><code>cohctl clear default-heap</code></router-link> - clears default heap for creating and starting clusters</p>

</li>
</ul>
<p><strong>Creating and managing profiles</strong></p>

<ul class="ulist">
<li>
<p><router-link to="#set-profile" @click.native="this.scrollFix('#set-profile')"><code>cohctl set profile</code></router-link> - set a profile value for creating and starting clusters</p>

</li>
<li>
<p><router-link to="#remove-profile" @click.native="this.scrollFix('#remove-profile')"><code>cohctl remove profile</code></router-link> - removes a profile value from the list of profile</p>

</li>
<li>
<p><router-link to="#get-profiles" @click.native="this.scrollFix('#get-profiles')"><code>cohctl get profiles</code></router-link> - displays the profiles that have been created</p>

</li>
</ul>

<h4 id="create-cluster">Create Cluster</h4>
<div class="section">
<p>The 'create cluster' command creates a local cluster, adds to the cohctl.yaml file
and starts it. You must have the 'mvn' executable and 'java' 17+ executable in your PATH for
this to work. This cluster is only for development/testing purposes and should not be used,
and is not supported in a production capacity. Supported versions are: CE 22.06 and above and
commercial 14.1.1.2206.1 and above. Default version is currently CE 22.06.9.
NOTE: This is an experimental feature and my be altered or removed in the future.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl create cluster cluster-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -a, --additional string         additional comma separated Coherence artifacts or others in G:A:V format
      --cache-config string       cache configuration file
  -p, --cluster-port int32        cluster port (default 7574)
  -C, --commercial                use commercial Coherence groupID (default CE)
  -e, --health-port int32         starting port for health
  -M, --heap-memory string        heap memory to allocate for JVM if default-heap not set (default "128m")
  -h, --help                      help for cluster
  -H, --http-port int32           http management port (default 30000)
  -j, --jmx-host string           remote JMX RMI host for management member
  -J, --jmx-port int32            remote JMX port for management member
  -L, --log-destination string    root directory to place log files in
  -l, --log-level int32           coherence log level (default 5)
  -t, --metrics-port int32        starting port for metrics
      --override-config string    override override file
  -T, --partition-count int32     partition count (default 257)
  -s, --persistence-mode string   persistence mode [on-demand active active-backup active-async] (default "on-demand")
  -P, --profile string            profile to add to cluster startup command line
  -F, --profile-first             only apply profile to the first member starting
  -r, --replicas int32            number of replicas (default 3)
  -I, --skip-deps                 skip pulling artifacts
  -S, --start-class string        class name to start server with (default com.tangosol.net.Coherence)
  -D, --startup-delay string      startup delay in millis for each server (default "0ms")
  -v, --version string            cluster version (default "22.06.9")
  -K, --wka string                well known address (default "127.0.0.1")
  -y, --yes                       automatically confirm the operation</pre>
</div>

<div class="admonition note">
<p class="admonition-inline">The log files are stored under the <code>.cohctl/logs/</code> directory off your home directory. You can
change this for an individual cluster by specifying the <code>-L</code> option when you create the cluster.</p>
</div>
<p><strong>Examples</strong></p>

<p>Add and start a cluster using all default values.</p>

<div class="admonition note">
<p class="admonition-inline">After the cluster has been created, the current context is automatically set to the new cluster.</p>
</div>
<markup
lang="bash"

>cohctl create cluster local</markup>

<p>Output:</p>

<markup
lang="bash"

>Cluster name:           local
Cluster version:        22.06.9
Cluster port:           7574
Partition count:        257
Management port:        30000
Server count:           3
Initial memory:         128m
Persistence mode:       on-demand
Group ID:               com.oracle.coherence.ce
Additional artifacts:
Log destination root:
Dependency tool:        mvn
Are you sure you want to create the cluster with the above details? (y/n) y

Checking 3 Maven dependencies...
- com.oracle.coherence.ce:coherence:22.06.9
- com.oracle.coherence.ce:coherence-json:22.06.9
- org.jline:jline:3.20.0
Starting 3 cluster members for cluster local
Starting cluster member storage-0...
Starting cluster member storage-1...
Starting cluster member storage-2...
Cluster added and started
Current context is now local</markup>

<p>Display the cluster members</p>

<markup
lang="bash"

>cohctl get members</markup>

<p>Output:</p>

<markup
lang="bash"

>Using cluster connection 'local' from current context.

Total cluster members: 3
Cluster Heap - Total: 384 MB Used: 123 MB Available: 261 MB (68.0%)
Storage Heap - Total: 384 MB Used: 123 MB Available: 261 MB (68.0%)

NODE ID  ADDRESS     PORT   PROCESS  MEMBER     ROLE             STORAGE  MAX HEAP  USED HEAP  AVAIL HEAP
      1  /127.0.0.1  61565    63754  storage-1  CoherenceServer  true       128 MB      28 MB      100 MB
      2  /127.0.0.1  61566    63753  storage-0  CoherenceServer  true       128 MB      25 MB      103 MB
      3  /127.0.0.1  61567    63755  storage-2  CoherenceServer  true       128 MB      70 MB       58 MB</markup>

<div class="admonition note">
<p class="admonition-inline">By default, Coherence CE groupId is used and the version is 22.06.9. You can change this via using <code>-C</code> for commercial and <code>-v</code> to change the Coherence version.</p>
</div>
<div class="admonition note">
<p class="admonition-inline">Additional dependencies <code>coherence-json</code> is included to enable Management over REST and <code>jline</code> is included for <code>CohQL</code> history support.</p>
</div>
<p>Add and start a commercial Coherence cluster (14.1.1.2206.5) and set initial memory for each cluster to 1g and use active persistence mode.</p>

<markup
lang="bash"

>cohctl create cluster local -C -v 14.1.1-2206-5 -M 1g -P active</markup>

<p>Output:</p>

<markup
lang="bash"

>Cluster name:         local
Cluster version:      14.1.1-2206-5
Cluster port:         7574
Partition count:      257
Management port:      30000
Server count:         3
Initial memory:       1g
Persistence mode:     active
Group ID:             com.oracle.coherence
Additional artifacts:
Log destination root:
Dependency tool:      mvn
Are you sure you want to create the cluster with the above details? (y/n) y

Skipping downloading Maven artifcts
Starting 3 cluster members for cluster local
Starting cluster member storage-0...
Starting cluster member storage-1...
Starting cluster member storage-2...
Cluster added and started</markup>

<div class="admonition note">
<p class="admonition-inline">In this example we are using the <code>-I</code> option to skip downloading maven artifacts as we know they are already installed locally.</p>
</div>
<p>Add and start a cluster using all default values but include additional <code>coherence-rest</code> and opentracing dependencies.</p>

<markup
lang="bash"

>cohctl create cluster local -a coherence-rest,io.opentracing:opentracing-api:0.33.0,io.opentracing:opentracing-util:0.33.0</markup>

<p>Output:</p>

<markup
lang="bash"

>Cluster name:           local
Cluster version:        22.06.9
Cluster port:           7574
Partition count:        257
Management port:        30000
Server count:           3
Initial memory:         128m
Persistence mode:       on-demand
Group ID:               com.oracle.coherence.ce
Additional artifacts:   coherence-rest,io.opentracing:opentracing-api:0.33.0,io.opentracing:opentracing-util:0.33.0
Log destination root:
Dependency tool:        mvn
Are you sure you want to create the cluster with the above details? (y/n) y

Checking 6 Maven dependencies...
- com.oracle.coherence.ce:coherence:22.06.9
- com.oracle.coherence.ce:coherence-json:22.06.9
- com.oracle.coherence.ce:coherence-rest:22.06.9
- io.opentracing:opentracing-api:0.33.0
- io.opentracing:opentracing-util:0.33.0
- org.jline:jline:3.20.0
Starting 3 cluster members for cluster local
Starting cluster member storage-0...
Starting cluster member storage-1...
Starting cluster member storage-2...
Cluster added and started with process ids: [3324 3330 3331]</markup>

</div>

<h4 id="scale-cluster">Scale Cluster</h4>
<div class="section">
<p>The 'scale cluster' command scales a cluster that was manually created.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl scale cluster [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -B, --backup-logs            backup old cache server log files
  -e, --health-port int32      starting port for health
  -M, --heap-memory string     heap memory to allocate for JVM if default-heap not set (default "128m")
  -h, --help                   help for cluster
  -l, --log-level int32        coherence log level (default 5)
  -t, --metrics-port int32     starting port for metrics
  -P, --profile string         profile to add to cluster startup command line
  -r, --replicas int32         number of replicas (default 3)
  -S, --start-class string     class name to start server with (default com.tangosol.net.Coherence)
  -D, --startup-delay string   startup delay in millis for each server (default "0ms")</pre>
</div>

<markup
lang="bash"

>cohctl scale cluster local -r 4</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to scale the cluster local up by 1 member(s) to 4 members? (y/n) y
Starting cluster member storage-3...
Cluster local scaled</markup>

<div class="admonition note">
<p class="admonition-inline">It is not yet supported to scale down a cluster.</p>
</div>
</div>

<h4 id="stop-cluster">Stop Cluster</h4>
<div class="section">
<p>The 'stop cluster' command stops a cluster that was manually created or started.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl stop cluster [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for cluster
  -y, --yes    automatically confirm the operation</pre>
</div>

<markup
lang="bash"

>cohctl stop cluster local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to stop 3 members for the cluster local? (y/n) y
killed process 47760
killed process 47761
killed process 47762
3 processes were stopped for cluster local</markup>

</div>

<h4 id="start-cluster">Start Cluster</h4>
<div class="section">
<p>The 'start cluster' command starts a cluster that was manually created.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl start cluster [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -B, --backup-logs            backup old cache server log files
  -e, --health-port int32      starting port for health
  -M, --heap-memory string     heap memory to allocate for JVM if default-heap not set (default "128m")
  -h, --help                   help for cluster
  -j, --jmx-host string        remote JMX RMI host for management member
  -J, --jmx-port int32         remote JMX port for management member
  -l, --log-level int32        coherence log level (default 5)
  -t, --metrics-port int32     starting port for metrics
  -P, --profile string         profile to add to cluster startup command line
  -F, --profile-first          only apply profile to the first member starting
  -r, --replicas int32         number of replicas (default 3)
  -S, --start-class string     class name to start server with (default com.tangosol.net.Coherence)
  -D, --startup-delay string   startup delay in millis for each server (default "0ms")</pre>
</div>

<p><strong>Examples</strong></p>

<p>Start a cluster and specify heap size of <code>1g</code>. (default is 128m)</p>

<markup
lang="bash"

>cohctl start cluster local -M 1g</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to start 3 members for cluster local? (y/n) y
Starting cluster member storage-0...
Starting cluster member storage-1...
Starting cluster member storage-2...
Cluster local and started</markup>

<p>Start a cluster and specify heap size of <code>1g</code> with 4 replicas/members.</p>

<markup
lang="bash"

>cohctl start cluster local -M 1g -r 4</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to start 4 members for cluster local? (y/n) y
Starting cluster member storage-0...
Starting cluster member storage-1...
Starting cluster member storage-2...
Starting cluster member storage-3...
Cluster local and started</markup>

<p>If you wish to enable remote RMI management on you cluster, as well as HTTP management,
you will need to use the following:</p>

<markup
lang="bash"

>cohctl start cluster local -J 9999 -j hostname</markup>

<ul class="ulist">
<li>
<p><code>-J</code> is the rmi port</p>

</li>
<li>
<p><code>-j</code> is the rmi host. You should set this to the hostname the cluster is running on. It will default to WKA address if not set.</p>

</li>
</ul>
</div>

<h4 id="start-console">Start Console</h4>
<div class="section">
<p>The 'start console' command starts a console client which connects to a
cluster using the current context or a cluster specified by using '-c'.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl start console [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -M, --heap-memory string   heap memory to allocate for JVM if default-heap not set (default "128m")
  -h, --help                 help for console
  -l, --log-level int32      coherence log level (default 5)
  -P, --profile string       profile to add to cluster startup command line</pre>
</div>

<p>Start a Coherence console.</p>

<markup
lang="bash"

>cohctl start console -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Starting client com.tangosol.net.CacheFactory...
2022-08-29 16:00:01.346/0.620 Oracle Coherence 22.06.9 <Info> (thread=main, member=n/a): Loaded operational configuration from "jar:file:/Users/user/.m2/repository/com/oracle/coherence/ce/coherence/22.06.9/coherence-22.06.9.jar!/tangosol-coherence.xml"
...

Map (?):</markup>

<div class="admonition note">
<p class="admonition-inline">Use <code>bye</code> to quit the console.</p>
</div>
</div>

<h4 id="start-cohql">Start CohQL</h4>
<div class="section">
<p>The 'start cohql' command starts a CohQL client which connects to a
cluster using the current context or a cluster specified by using '-c'..</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl start cohql [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -X, --extend               start CohQL as Extend client. Only works for default cache config
  -f, --file string          file name to read CohQL commands from
  -M, --heap-memory string   heap memory to allocate for JVM if default-heap not set (default "128m")
  -h, --help                 help for cohql
  -l, --log-level int32      coherence log level (default 5)
  -P, --profile string       profile to add to cluster startup command line
  -S, --statement string     statement to execute enclosed in double quotes</pre>
</div>

<p>Start a CohQL session.</p>

<markup
lang="bash"

>cohctl start cohql -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Starting client com.tangosol.coherence.dslquery.QueryPlus...
Coherence Command Line Tool

CohQL&gt;</markup>

<div class="admonition note">
<p class="admonition-inline">Use <code>bye</code> to quit the console.</p>
</div>
<p>Start a CohQL Session and execute a statement</p>

<markup
lang="bash"

>cohctl start cohql -c local -S "insert into test key(1) value(1)"</markup>

<p>Start a CohQL Session and execute statements from a file.</p>

<markup
lang="bash"

>cohctl start cohql -c local -f /tmp/run.cohql</markup>

</div>

<h4 id="start-class">Start Class</h4>
<div class="section">
<p>The 'start class' command starts a specific Java class which connects to a
cluster using the current context or a cluster specified by using '-c'.
The class name must include the full package and class name and must be included in
an artefact included in the initial cluster creation.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl start class [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -X, --extend               start a class as Extend client. Only works for default cache config
  -M, --heap-memory string   heap memory to allocate for JVM if default-heap not set (default "128m")
  -h, --help                 help for class
  -l, --log-level int32      coherence log level (default 5)
  -P, --profile string       profile to add to cluster startup command line</pre>
</div>

<markup
lang="bash"

>cohctl start class com.my.company.Class -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Starting client com.my.company.Class...</markup>

</div>

<h4 id="set-use-gradle">Set Use Gradle</h4>
<div class="section">
<p>The 'set use-gradle' command sets whether to use gradle for dependency management.
This setting only affects when you create a cluster. If set to false, the default of Maven will be used.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl set use-gradle {true|false} [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for use-gradle</pre>
</div>

<markup
lang="bash"

>cohctl set use-gradle true</markup>

<p>Output:</p>

<markup
lang="bash"

>Use Gradle is now set to true</markup>

</div>

<h4 id="get-use-gradle">Get Use Gradle</h4>
<div class="section">
<p>The 'get use-gradle' command displays the current setting for whether to
use gradle for dependency management. If set to false, the default of Maven is used.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get use-gradle [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for use-gradle</pre>
</div>

<markup
lang="bash"

>cohctl get use-gradle</markup>

<p>Output:</p>

<markup
lang="bash"

>Use Gradle: true</markup>

</div>

<h4 id="set-default-heap">Set Default Heap</h4>
<div class="section">
<p>The 'set default-heap' command sets the default heap when creating and starting cluster.
Valid values are in the format suitable for -Xms such as 256m, 1g, etc.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl set default-heap value [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for default-heap</pre>
</div>

<p>Set default heap to 512m.</p>

<markup
lang="bash"

>cohctl set default-heap 512m</markup>

<p>Output:</p>

<markup
lang="bash"

>Default heap is now set to 512m</markup>

</div>

<h4 id="get-default-heap">Get Default Heap</h4>
<div class="section">
<p>The 'get default-heap' displays the default heap for creating and starting clusters.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get default-heap [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for default-heap</pre>
</div>

<markup
lang="bash"

>cohctl get default-heap</markup>

<p>Output:</p>

<markup
lang="bash"

>Current default heap: 512m</markup>

</div>

<h4 id="clear-default-heap">Clear Default Heap</h4>
<div class="section">
<p>The 'clear default-heap' clears the default heap for creating and starting clusters.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl clear default-heap [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for default-heap</pre>
</div>

<markup
lang="bash"

>cohctl clear default-heap</markup>

<p>Output:</p>

<markup
lang="bash"

>Default heap has been cleared</markup>

<div class="admonition note">
<p class="admonition-inline">If no default-heap is set, then the default of 128m is used unless a value for <code>-M</code> is specified.</p>
</div>
</div>

<h4 id="set-profile">Set Profile</h4>
<div class="section">
<p>The 'set profile' command sets a profile value for creating and starting clusters.
Profiles can be specified using the '-P' option when creating and starting clusters. They
contain property values to be set prior to the final class and must be surrounded by quotes
and be space delimited. If you set a profile that exists, it will be overwritten.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl set profile profile-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help           help for profile
  -v, --value string   profile value to set
  -y, --yes            automatically confirm the operation</pre>
</div>

<markup
lang="bash"

>cohctl set profile debug-enabled -v "-Dmy.debug.enabled=true -Dmy.debug.level=10"</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to set the profile debug-enabled to a value of [-Dmy.debug.enabled=true -Dmy.debug.level=10]? (y/n) y
profile debug-enabled was set to value [-Dmy.debug.enabled=true -Dmy.debug.level=10]</markup>

<p>When you have set the profile you can startup or create a cluster using that profile by
specifying <code>-P profile-name</code> for the <code>cohctl start cluster</code> or <code>cohctl create cluster</code> commands.</p>

</div>

<h4 id="remove-profile">Remove Profile</h4>
<div class="section">
<p>The 'remove profile' command removes a profile value from the list of profiles.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl remove profile profile-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for profile
  -y, --yes    automatically confirm the operation</pre>
</div>

<markup
lang="bash"

>cohctl remove profile debug-enabled</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to remove the profile debug-enabled? (y/n) y
profile debug-enabled was removed</markup>

</div>

<h4 id="get-profiles">Get Profiles</h4>
<div class="section">
<p>The 'get profiles' displays the profiles that have been created.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get profiles [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for profiles</pre>
</div>

<markup
lang="bash"

>cohctl get profiles</markup>

<p>Output:</p>

<markup
lang="bash"

>PROFILE    VALUE
profile1   -Dproperty1.value=2
profile2   -Dproperty2.value=2 -Dproperty3.value=4</markup>

</div>
</div>

<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/reference/05_clusters">Clusters</router-link></p>

</li>
</ul>
</div>
</div>
</doc-view>
