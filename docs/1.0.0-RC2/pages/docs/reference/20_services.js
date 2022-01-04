<doc-view>

<h2 id="_services">Services</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>There are various commands that allow you to work with and manage cluster services.</p>

<ul class="ulist">
<li>
<p><router-link to="#get-services" @click.native="this.scrollFix('#get-services')"><code>cohctl get services</code></router-link> - displays the services for a cluster</p>

</li>
<li>
<p><router-link to="#describe-service" @click.native="this.scrollFix('#describe-service')"><code>cohctl describe service</code></router-link> - shows information related to a specific service</p>

</li>
<li>
<p><router-link to="#start-service" @click.native="this.scrollFix('#start-service')"><code>cohctl start service</code></router-link> - starts a specific service on a cluster member</p>

</li>
<li>
<p><router-link to="#stop-service" @click.native="this.scrollFix('#stop-service')"><code>cohctl stop service</code></router-link> - forces a specific service to stop on a cluster member</p>

</li>
<li>
<p><router-link to="#shutdown-service" @click.native="this.scrollFix('#shutdown-service')"><code>cohctl shutdown service</code></router-link> - performs a controlled shut-down of a specific service on a cluster member</p>

</li>
<li>
<p><router-link to="#set-service" @click.native="this.scrollFix('#set-service')"><code>cohctl set service</code></router-link> - sets a service attribute across one or more members</p>

</li>
</ul>

<h4 id="get-services">Get Services</h4>
<div class="section">
<p>The 'get services' command displays services for a cluster using various options.
You may specify the service type as well a status-ha value to wait for. You
can also specify '-o wide' to display addition information.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get services [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help               help for services
  -a, --status-ha string   StatusHA to wait for. Used in conjunction with -T option (default "none")
  -T, --timeout int32      Timeout to wait for StatusHA value of all services (default 60)
  -t, --type string        Service types to show. E.g. DistributedCache, FederatedCache,
                           Invocation, Proxy, RemoteCache or ReplicatedCache (default "all")</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display all services.</p>

<markup
lang="bash"

>$ cohctl get services -c local</markup>

<p>Display all services of type <code>DistributedCache</code></p>

<markup
lang="bash"

>$ cohctl get services -c local -t DistributedCache</markup>

<p>Watch all services of type <code>DistributedCache</code></p>

<markup
lang="bash"

>$ cohctl get services -c local -t DistributedCache -w</markup>

<p>Wait for all services of type <code>DistributedCache</code> to become <code>MACHINE-SAFE</code>.</p>

<markup
lang="bash"

>$ cohctl get services -c local -t DistributedCache -w -a MACHINE-SAFE</markup>

</div>

<h4 id="describe-service">Describe Service</h4>
<div class="section">
<p>The 'describe service' command shows information related to services. This
includes information about each service member as well as Persistence information if the
service is a cache service.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl describe service service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for service</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl describe service PartitionedCache -c local</markup>

</div>

<h4 id="start-service">Start Service</h4>
<div class="section">
<p>The 'start service' command starts a specific service on a cluster member.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl start service service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help          help for service
  -n, --node string   node id to target
  -y, --yes           Automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl start service PartitionedCache -n 1 -c local

Are you sure you want to perform start for service PartitionedCache on node 1? (y/n) y
operation completed</markup>

</div>

<h4 id="stop-service">Stop Service</h4>
<div class="section">
<p>The 'stop service' command forces a specific service to stop on a cluster member.
Use the shutdown service command for normal service termination.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl stop service service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help          help for service
  -n, --node string   node id to target
  -y, --yes           Automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl stop service PartitionedCache -n 1 -c local

Are you sure you want to perform stop for service PartitionedCache on node 1? (y/n) y
operation completed</markup>

</div>

<h4 id="shutdown-service">Shutdown Service</h4>
<div class="section">
<p>The 'shutdown service' command performs a controlled shut-down of a specific service
on a cluster member. Shutting down a service is preferred over stopping a service.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl shutdown service service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help          help for service
  -n, --node string   node id to target
  -y, --yes           Automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl shutdown service PartitionedCache -n 1 -c local

Are you sure you want to perform shutdown for service PartitionedCache on node 1? (y/n) y
operation completed</markup>

</div>

<h4 id="set-service">Set Service</h4>
<div class="section">
<p>The 'set service' command sets an attribute for a service across one or member nodes.
The following attribute names are allowed: threadCount, threadCountMin, threadCountMax or
taskHungThresholdMillis or requestTimeoutMillis.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl set service &lt;service-name&gt; [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -a, --attribute string   attribute name to set
  -h, --help               help for service
  -n, --node string        comma separated node ids to target (default "all")
  -v, --value string       attribute value to set
  -y, --yes                Automatically confirm the operation</pre>
</div>

<p>See the <a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.0/manage/oracle-coherence-mbeans-reference.html">Service MBean Reference</a>
for more information on the above attributes.</p>

<p><strong>Examples</strong></p>

<p>Set the thread count min for a service.</p>

<p>List the services and inspect the current thread count min.</p>

<markup
lang="bash"

>$ cohctl get services -c local

SERVICE NAME         TYPE              MEMBERS  STATUS HA  STORAGE  PARTITIONS
Proxy                Proxy                   2  n/a             -1          -1
PartitionedTopic     DistributedCache        2  NODE-SAFE        2         257
PartitionedCache2    DistributedCache        2  NODE-SAFE        2         257
PartitionedCache     DistributedCache        2  NODE-SAFE        2         257
ManagementHttpProxy  Proxy                   1  n/a             -1          -1

$ cohctl get services -o  jsonpath="$.items[?(@.name == 'PartitionedCache')]..['nodeId','name','threadCountMin']"
["2","PartitionedCache",1,"1","PartitionedCache",1]</markup>

<div class="admonition note">
<p class="admonition-inline">The above shows that the <code>threadCountMin</code> is 1 for both nodes.</p>
</div>
<p>Set the <code>threadCountMin</code> to 10 for each service member.</p>

<markup
lang="bash"

>$ cohctl set service PartitionedCache -a threadCountMin -v 10 -c local

Selected service: PartitionedCache
Are you sure you want to set the value of attribute threadCountMin to 10 for all 2 nodes? (y/n) y
operation completed

$ cohctl get services -o  jsonpath="$.items[?(@.name == 'PartitionedCache')]..['nodeId','name','threadCountMin']"
["2","PartitionedCache",10,"1","PartitionedCache",10]</markup>

</div>
</div>

<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/examples/05_rolling_restarts">Rolling Restarts</router-link></p>

</li>
<li>
<p><router-link to="/docs/reference/25_caches">Caches</router-link></p>

</li>
</ul>
</div>
</div>
</doc-view>
