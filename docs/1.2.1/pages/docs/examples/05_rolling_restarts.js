<doc-view>

<h2 id="_rolling_restarts">Rolling Restarts</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>This example walks you through how to monitor the High Available (HA) Status or <code>StatusHA</code>
value for Coherence Partitioned Services within a cluster by using the <code>cohctl get services</code> command.</p>

<p><code>StatusHA</code> is most commonly used to ensure services are in a
safe state between restarting cache servers during a rolling restart.</p>

</div>

<h3 id="_setup_for_this_example">Setup for this Example</h3>
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
A Coherence console client running

</li>
</ol>
</div>

<h3 id="_run_the_example">Run the example</h3>
<div class="section">
<p>In this example we will carry out a rolling restart of our cluster to simulate applying an application code patch to
our cluster. For more details on rolling restarts, please see <a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.0/develop-applications/starting-and-stopping-cluster-members.html">Starting and Stopping Cluster Members</a> in the Coherence documentation.</p>

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


<h4 id="_1_show_the_clusters">1. Show the clusters</h4>
<div class="section">
<markup
lang="bash"

>$ cohctl get clusters
CONNECTION  TYPE  URL                                                  VERSION  CLUSTER NAME  CLUSTER TYPE
local       http  http://localhost:30000/management/coherence/cluster  21.12    my-cluster    Standalone

$ cohctl set context local
Current context is now local</markup>

</div>

<h4 id="_2_get_the_members">2. Get the members</h4>
<div class="section">
<markup
lang="bash"

>$ cohctl get members -o wide
Using cluster connection 'local' from current context.

Cluster Heap - Total: 6.750GB, Used: 1.076GB, Available: 5.674GB (84.1%)

NODE ID  ADDRESS          PORT  PROCESS  MEMBER  ROLE              MACHINE   RACK  SITE  PUBLISHER  RECEIVER  MAX HEAP  USED HEAP  AVAIL HEAP
      1  /192.168.1.124  58374    42988  n/a     Management        n/a       n/a   n/a       0.995     1.000     512MB       53MB       459MB
      2  /192.168.1.124  58389    43011  n/a     CoherenceServer   machine1  n/a   n/a       1.000     1.000   1.000GB      307MB       717MB
      3  /192.168.1.124  58399    43033  n/a     CoherenceServer   machine1  n/a   n/a       0.997     1.000   1.000GB      140MB       884MB
      4  /192.168.1.124  58434    43055  n/a     CoherenceServer   machine2  n/a   n/a       0.997     1.000   1.000GB      175MB       849MB
      5  /192.168.1.124  58464    43081  n/a     CoherenceServer   machine2  n/a   n/a       0.997     1.000   1.000GB      184MB       840MB
      7  /192.168.1.124  58774    44276  n/a     CoherenceServer   machine3  n/a   n/a       1.000     1.000   1.000GB      124MB       900MB
      8  /192.168.1.124  58808    44473  n/a     CoherenceServer   machine3  n/a   n/a       1.000     1.000   1.000GB       97MB       927MB
      9  /192.168.1.124  58868    44523  n/a     CoherenceConsole  n/a       n/a   n/a       1.000     1.000     256MB       22MB       234MB</markup>

<div class="admonition note">
<p class="admonition-inline">We can see the management node on Node 1, the storage members on nodes 2-5 and the console on node 6.</p>
</div>
</div>

<h4 id="_3_get_the_partitioned_services">3. Get the partitioned services</h4>
<div class="section">
<markup
lang="bash"

>$ cohctl get services -t DistributedCache -o wide
Using cluster connection 'local' from current context.

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
</div>

<h4 id="_4_view_the_caches">4. View the caches</h4>
<div class="section">
<p>In our case we have the following caches defined:</p>

<markup
lang="bash"

>$ cohctl get caches
Using cluster connection 'local' from current context.

Total Caches: 3, Total primary storage: 175MB

SERVICE            CACHE   CACHE SIZE        BYTES     MB
PartitionedCache   tim          1,000   10,160,000    9MB
PartitionedCache2  test-1     100,000  116,000,000  110MB
PartitionedCache2  test-2      50,000   58,000,000   55MB</markup>

<div class="admonition note">
<p class="admonition-inline">You can use the <code>-o wide</code> to display more information.</p>
</div>
</div>

<h4 id="_5_start_watching_the_services">5. Start watching the services</h4>
<div class="section">
<markup
lang="bash"

>$ cohctl get services -t DistributedCache -w -o wide</markup>

<div class="admonition note">
<p class="admonition-inline">The above will continue watching the services. Keep this open in a separate terminal.</p>
</div>
</div>

<h4 id="_6_carry_out_a_rolling_restart_of_the_cluster">6. Carry out a rolling restart of the cluster.</h4>
<div class="section">
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
</div>

<h3 id="_scripting_the_rolling_redeploy">Scripting the Rolling Redeploy</h3>
<div class="section">
<p>The Coherence CLI cannot directly start or stop members, but can be use in scripts to detect when services have reached a certain state.</p>

<p>You can use the <code>-a MACHINE-SAFE</code> option of <code>get services</code> to wait up to the timeout value (default to 60 seconds), for the StatusHA
to be equal or greater that the value you specified. If it reaches this value in the timeout, the command will return 0 exit code but if
it does not, then a return code of 1 is returned.</p>

<p>The following example would wait up to 60 seconds for DistributedCache services to be MACHINE-SAFE.</p>

<markup
lang="bash"

>$ cohctl get services -t DistributedCache -w -a MACHINE-SAFE</markup>

</div>
</div>

<h2 id="_see_also">See Also</h2>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/reference/20_services">Services</router-link></p>

</li>
<li>
<p><a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.0/develop-applications/starting-and-stopping-cluster-members.html">Starting and Stopping Cluster Members</a></p>

</li>
<li>
<p><a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.0/manage/oracle-coherence-mbeans-reference.html">Coherence MBean Reference</a></p>

</li>
</ul>
</div>
</doc-view>
