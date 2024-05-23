<doc-view>

<h2 id="_rolling_restarts">Rolling Restarts</h2>
<div class="section">
<p>The Coherence CLI allows you to check your cluster state during rolling restarts. This is important as you
do now what to continue your rolling restart until you are sure all data is safe.</p>

<p>The CLI provides a number of ways to do this depending upon how your cluster is setup.</p>

<ul class="ulist">
<li>
<p><router-link to="#get-services" @click.native="this.scrollFix('#get-services')">Using "cohctl get services"</router-link> - Use this option if you have management over REST enabled</p>

</li>
<li>
<p><router-link to="#monitor-health" @click.native="this.scrollFix('#monitor-health')">Using"cohctl monitor health"</router-link> - Use this option if you have Health endpoints enabled</p>

</li>
</ul>

<h3 id="get-services">Checking StatusHA with "cohctl get services"</h3>
<div class="section">
<p>This example walks you through how to monitor the High Available (HA) Status or <code>StatusHA</code>
value for Coherence Partitioned Services within a cluster by using the <code>cohctl get services</code> command.</p>

<p><code>StatusHA</code> is most commonly used to ensure services are in a
safe state between restarting cache servers during a rolling restart.</p>


<h4 id="_setup_for_this_example">Setup for this Example</h4>
<div class="section">
<p>In this example we have a cluster called <code>my-cluster</code> with the following setup:</p>

<ol style="margin-left: 15px;">
<li>
A single storage-disabled management node running Management over REST enabled

</li>
<li>
2 storage-enabled nodes on <code>machine1</code>

</li>
<li>
2 storage-enabled nodes on <code>machine2</code>

</li>
<li>
2 storage-enabled nodes on <code>machine3</code>

</li>
<li>
A Coherence console client

</li>
</ol>
</div>

<h4 id="_run_the_example">Run the example</h4>
<div class="section">
<p>In this example we will carry out a rolling restart of our cluster to simulate applying an application code patch to
our cluster. For more details on rolling restarts, please see <a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/develop-applications/starting-and-stopping-cluster-members.html">Starting and Stopping Cluster Members</a> in the Coherence documentation.</p>

<p>The process will be:</p>

<ol style="margin-left: 15px;">
<li>
Stop member 1 on first machine

</li>
<li>
Wait for NODE-SAFE - (Can&#8217;t get to MACHINE-SAFE because of unbalanced partition counts between machines)

</li>
<li>
Stop member 2 on first machine

</li>
<li>
Wait for MACHINE-SAFE - When they could apply an application patch to our first machine.

</li>
<li>
Start member 1 and 2 on first machine

</li>
<li>
Wait for MACHINE-SAFE

</li>
<li>
Repeat steps 1-6 on second and third machines

</li>
</ol>
<p>Read on below for the example.</p>

<p><strong>1. Show the clusters</strong></p>

<markup
lang="bash"

>cohctl get clusters</markup>

<p>Output:</p>

<markup
lang="bash"

>CONNECTION  TYPE  URL                                                  VERSION  CLUSTER NAME  CLUSTER TYPE  LOCAL
local       http  http://localhost:30000/management/coherence/cluster  23.03    my-cluster    Standalone    false</markup>

<markup
lang="bash"

>cohctl set context local</markup>

<p>Output:</p>

<markup
lang="bash"

>Current context is now local</markup>

<p><strong>2. Get the members</strong></p>

<markup
lang="bash"

>cohctl get members -o wide -m</markup>

<p>Output:</p>

<markup
lang="bash"

>Using cluster connection 'local' from current context.

Cluster Heap - Total: 6.750GB, Used: 1.076GB, Available: 5.674GB (84.1%)

NODE ID  ADDRESS          PORT  PROCESS  MEMBER  ROLE              MACHINE   RACK  SITE  PUBLISHER  RECEIVER  MAX HEAP  USED HEAP  AVAIL HEAP
      1  /192.168.1.124  58374    42988  n/a     Management        n/a       n/a   n/a       0.995     1.000    512 MB      53 MB      459 MB
      2  /192.168.1.124  58389    43011  n/a     CoherenceServer   machine1  n/a   n/a       1.000     1.000   1024 MB     307 MB      717 MB
      3  /192.168.1.124  58399    43033  n/a     CoherenceServer   machine1  n/a   n/a       0.997     1.000   1024 MB     140 MB      884 MB
      4  /192.168.1.124  58434    43055  n/a     CoherenceServer   machine2  n/a   n/a       0.997     1.000   1024 MB     175 MB      849 MB
      5  /192.168.1.124  58464    43081  n/a     CoherenceServer   machine2  n/a   n/a       0.997     1.000   1024 MB     184 MB      840 MB
      7  /192.168.1.124  58774    44276  n/a     CoherenceServer   machine3  n/a   n/a       1.000     1.000   1024 MB     124 MB      900 MB
      8  /192.168.1.124  58808    44473  n/a     CoherenceServer   machine3  n/a   n/a       1.000     1.000   1024 MB      97 MB      927 MB
      9  /192.168.1.124  58868    44523  n/a     CoherenceConsole  n/a       n/a   n/a       1.000     1.000    256 M       22 MB      234 MB</markup>

<div class="admonition note">
<p class="admonition-inline">We can see the management node on Node 1, the storage members on nodes 2-5 and the console on node 6.</p>
</div>
<p><strong>3. Get the partitioned services</strong></p>

<markup
lang="bash"

>cohctl get services -t DistributedCache -o wide</markup>

<p>Output:</p>

<markup
lang="bash"

>Using cluster connection 'local' from current context.

SERVICE NAME       TYPE              MEMBERS  STATUS HA     STORAGE  PARTITIONS  ENDANGERED  VULNERABLE  UNBALANCED  STATUS
PartitionedTopic   DistributedCache        7  MACHINE-SAFE        6         257           0           0           0  Safe
PartitionedCache2  DistributedCache        7  MACHINE-SAFE        6         257           0           0           0  Safe
PartitionedCache   DistributedCache        7  MACHINE-SAFE        6         257           0           0           0  Safe</markup>

<p>See below for explanations of the above columns:</p>

<ul class="ulist">
<li>
<p>STATUS HA - The High Availability (HA) status for this service. A value of MACHINE-SAFE indicates that all the cluster members running on any given computer could be stopped without data loss. A value of NODE-SAFE indicates that a cluster member could be stopped without data loss. A value of ENDANGERED indicates that abnormal termination of any cluster member that runs this service may cause data loss. A value of N/A indicates that the service has no high availability impact.</p>

</li>
<li>
<p>STORAGE - Specifies the total number of cluster members running this service for which local storage is enabled</p>

</li>
<li>
<p>PARTITIONS - The total number of partitions that every cache storage is divided into</p>

</li>
<li>
<p>ENDANGERED - The total number of partitions that are not currently backed up</p>

</li>
<li>
<p>VULNERABLE - The total number of partitions that are backed up on the same machine where the primary partition owner resides</p>

</li>
<li>
<p>UNBALANCED - The total number of primary and backup partitions that remain to be transferred until the partition distribution across the storage enabled service members is fully balanced</p>

</li>
</ul>
<p><strong>4. View the caches</strong></p>

<p>In our case we have the following caches defined:</p>

<markup
lang="bash"

>cohctl get caches -m</markup>

<p>Output:</p>

<markup
lang="bash"

>Using cluster connection 'local' from current context.

Total Caches: 3, Total primary storage: 175MB

SERVICE            CACHE      COUNT    SIZE
PartitionedCache   tim        1,000    9 MB
PartitionedCache2  test-1   100,000  110 MB
PartitionedCache2  test-2    50,000   55 MB</markup>

<div class="admonition note">
<p class="admonition-inline">You can use the <code>-o wide</code> to display more information.</p>
</div>
<p><strong>5. Start watching the services</strong></p>

<markup
lang="bash"

>cohctl get services -t DistributedCache -w -o wide</markup>

<div class="admonition note">
<p class="admonition-inline">The above will continue watching the services. Keep this open in a separate terminal.</p>
</div>
<p><strong>6. Carry out a rolling restart of the cluster</strong></p>

<p>With the above command running in a separate terminal, carry out the following for each machine and watch for the StatusHA values.</p>

<ol style="margin-left: 15px;">
<li>
Stop member 1 on first machine

</li>
<li>
Wait for NODE-SAFE - When stopping the first cache server, you may observe the service StatusHA go to ENDANGERED straight after Coherence detects the failure and starts the rebalancing. When the StatusHA returns to NODE-SAFE, and unbalanced partitions are zero, you can continue.

</li>
<li>
Stop member 2 on first machine

</li>
<li>
Wait for MACHINE-SAFE - We will pretend to apply the software patch.

</li>
<li>
Start member 1 and 2 on first machine

</li>
<li>
Wait for MACHINE-SAFE

</li>
<li>
Repeat steps 1-6 on second and third machines

</li>
</ol>
</div>

<h4 id="_scripting_the_rolling_redeploy">Scripting the Rolling Redeploy</h4>
<div class="section">
<p>The Coherence CLI cannot directly start or stop members, but can be used in scripts to detect when services have reached a certain state.</p>

<p>You can use the <code>-a MACHINE-SAFE</code> option of <code>get services</code> to wait up to the timeout value (default to 60 seconds), for the StatusHA
to be equal or greater that the value you specified. If it reaches this value in the timeout, the command will return 0 exit code but if
it does not, then a return code of 1 is returned.</p>

<p>The following example would wait up to 60 seconds for DistributedCache services to be MACHINE-SAFE.</p>

<markup
lang="bash"

>cohctl get services -t DistributedCache -w -a MACHINE-SAFE</markup>

</div>
</div>

<h3 id="monitor-health">Checking health with "cohctl monitor health"</h3>
<div class="section">
<p>The <router-link :to="{path: '/docs/reference/90_health', hash: '#monitor-health'}">cohctl monitor health</router-link> command provides a different option to check for cluster health
if you have configured http health endpoints as described  <a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/manage/using-health-check-api.html">here</a>.</p>

<p>To use this option you must have configured the following:
* You are using Coherence CE version 22.06.+ or commercial version 14.1.1.2206.+
* You are starting coherence servers using <code>com.tangosol.net.Coherence</code></p>

<p>By default, if you start Coherence via <code>com.tangosol.net.Coherence</code>, the HTTP health port is ephemeral but you
can change by setting <code>-Dcoherence.health.http.port=your-port</code>.</p>


<h4 id="_setup_for_this_example_2">Setup for this Example</h4>
<div class="section">
<p>In this example we have a cluster with the following setup:</p>

<ul class="ulist">
<li>
<p>3 storage-enabled nodes running <code>com.tangosol.net.Coherence</code></p>

</li>
<li>
<p>A single Coherence console client to add data</p>

</li>
</ul>
</div>

<h4 id="_run_the_example_2">Run the example</h4>
<div class="section">
<p>In this example we will carry out a rolling restart of our cluster to simulate applying an application code patch to
our cluster. The process will be:</p>

<ol style="margin-left: 15px;">
<li>
Start all 3 members, console and add data

</li>
<li>
Start the health monitoring

</li>
<li>
Run the <code>cohctl monitor health</code> command

</li>
<li>
Stop member1

</li>
<li>
Wait for health to be stable

</li>
<li>
Restart member1 and wait for health to be stable

</li>
<li>
Repeat steps 4-6 on second and third member

</li>
</ol>
<p><strong>1. Start all 3 members, console and add data</strong></p>

<p>From the directory where your Coherence jar is, or by specifying the full path to coherence.jar, start the three cache servers
using the following:</p>

<markup
lang="bash"

>java -cp coherence.jar -Dcoherence.wka=127.0.0.1 com.tangosol.net.Coherence</markup>

<p>Start a console and add data.</p>

<markup
lang="bash"

>java -cp coherence.jar -Dcoherence.wka=127.0.0.1 -Dcoherence.distributed.storage=false com.tangosol.net.CacheFactory</markup>

<p>At the prompt type the following to add 100,000 entries:</p>

<markup
lang="bash"

>cache test
bulkput 100000 100 0 100
size</markup>

<div class="admonition note">
<p class="admonition-inline">You can leave the console open.</p>
</div>
<p><strong>2. Start the health monitoring</strong></p>

<p>The <code>-n</code> option specifies a cluster host/port to connect to to query the health endpoints.</p>

<markup
lang="bash"

>cohctl monitor health -n localhost:7574 -IW</markup>

<p>Output:</p>

<markup
lang="bash"

>2024-05-23 11:48:33.522017 +0800 AWST m=+5.509412654

HEALTH MONITORING
------------------
Name Service:    localhost:7574
Cluster Name:    timmiddleton's cluster
All Nodes Safe:  true

URL                      NODE ID  STARTED  LIVE  READY  SAFE  OVERALL
http://127.0.0.1:64307/      n/a      200   200    200   200        4
http://127.0.0.1:64328/      n/a      200   200    200   200        4
http://127.0.0.1:64329/      n/a      200   200    200   200        4</markup>

<div class="admonition note">
<p class="admonition-inline">The <code>-I</code> option ignores any errors connecting to the names service port and <code>-W</code> refreshes the screen.</p>
</div>
<p><strong>3. Carry out the rolling restart</strong></p>

<p>Repeat the following for each of the cache servers you started.</p>

<ul class="ulist">
<li>
<p>Stop member1 by using <code>CTRL-C</code></p>

</li>
<li>
<p>As the member exits, you will see <code>All Nodes Safe</code> become <code>false</code></p>

</li>
<li>
<p>Wait until <code>All Nodes Safe</code> becomes <code>true</code></p>

</li>
<li>
<p>Re-start the cache service simulating the updated application</p>

</li>
<li>
<p>Wait until <code>All Nodes Safe</code> becomes <code>true</code> again and repeat the above steps for all members</p>

</li>
</ul>
<p><strong>4. Verify the data using the console</strong></p>

<p>At the console prompt, type <code>size</code> to verify the number of cache entries is still 100,000.</p>

<p>Type <code>bye</code> to exit the console.</p>

</div>

<h4 id="_scripting_the_rolling_redeploy_2">Scripting the Rolling Redeploy</h4>
<div class="section">
<p>As mentioned above, the Coherence CLI cannot directly start or stop members, but can be
used in scripts to detect when services have reached a certain state.</p>

<p>The following example would wait up to 60 seconds for all nodes become safe.</p>

<p>If it reaches this value in the timeout, the command will return 0 exit code but if
it does not, then a return code of 1 is returned.</p>

<markup
lang="bash"

>cohctl monitor health -n localhost:7574 -T 60 -w</markup>

</div>
</div>
</div>

<h2 id="_see_also">See Also</h2>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/reference/20_services">Services</router-link></p>

</li>
<li>
<p><router-link to="/docs/reference/90_health">Health Commands</router-link></p>

</li>
<li>
<p><a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/develop-applications/starting-and-stopping-cluster-members.html">Starting and Stopping Cluster Members</a></p>

</li>
<li>
<p><a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/manage/oracle-coherence-mbeans-reference.html">Coherence MBean Reference</a></p>

</li>
<li>
<p><a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/manage/using-health-check-api.html">Coherence Health API</a></p>

</li>
</ul>
</div>
</doc-view>
