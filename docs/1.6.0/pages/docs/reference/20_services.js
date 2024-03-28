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
<p><router-link to="#get-service-storage" @click.native="this.scrollFix('#get-service-storage')"><code>cohctl get service-storage</code></router-link> - displays partitioned services storage information for a cluster</p>

</li>
<li>
<p><router-link to="#get-service-members" @click.native="this.scrollFix('#get-service-members')"><code>cohctl get service-members</code></router-link> - displays service members</p>

</li>
<li>
<p><router-link to="#get-service-distributions" @click.native="this.scrollFix('#get-service-distributions')"><code>cohctl get service-distributions</code></router-link> - displays partition distribution information for a service"</p>

</li>
<li>
<p><router-link to="#get-service-description" @click.native="this.scrollFix('#get-service-description')"><code>cohctl get service-description</code></router-link> - displays service description including membership"</p>

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
  -a, --status-ha string   statusHA to wait for. Used in conjunction with -T option (default "none")
  -T, --timeout int32      timeout to wait for StatusHA value of all services (default 60)
  -t, --type string        service types to show. E.g. DistributedCache, FederatedCache, PagedTopic,
                           Invocation, Proxy, RemoteCache or ReplicatedCache (default "all")</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display all services.</p>

<markup
lang="bash"

>cohctl get services -c local</markup>

<p>Display all services of type <code>DistributedCache</code></p>

<markup
lang="bash"

>cohctl get services -c local -t DistributedCache</markup>

<p>Watch all services of type <code>DistributedCache</code></p>

<markup
lang="bash"

>cohctl get services -c local -t DistributedCache -w</markup>

<p>Wait for all services of type <code>DistributedCache</code> to become <code>MACHINE-SAFE</code>.</p>

<markup
lang="bash"

>cohctl get services -c local -t DistributedCache -w -a MACHINE-SAFE</markup>

<div class="admonition note">
<p class="admonition-inline">If the above services does become machine safe in the timeout, the return code of the command will be zero, otherwise the return code will be 1.</p>
</div>
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

>cohctl describe service PartitionedCache -c local</markup>

</div>

<h4 id="get-service-storage">Get Service Storage</h4>
<div class="section">
<p>The 'get service-storage' command displays partitioned services storage for a cluster including
information regarding partition sizes.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get service-storage [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for service-storage</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl get service-storage -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>SERVICE NAME      PARTITIONS  NODES  AVG PARTITION  MAX PARTITION  AVG STORAGE  MAX STORAGE NODE  MAX NODE
"$SYS:Config"            257      6           0 MB           0 MB         0 MB              0 MB         -
PartitionedCache         257      6           0 MB           0 MB        18 MB             18 MB         2
PartitionedTopic         257      6           0 MB           0 MB         0 MB              0 MB         -</markup>

</div>

<h4 id="get-service-members">Get Service Members</h4>
<div class="section">
<p>The 'get service-members' command displays service members for a cluster.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get service-members service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -x, --exclude   exclude storage-disabled clients
  -h, --help      help for service-members
  -B, --include   include members with backlog only</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl get service-members PartitionedCache -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Service: PartitionedCache
NODE ID  THREADS  IDLE  THREAD UTIL  MIN THREADS    MAX THREADS  TASK COUNT  TASK BACKLOG  PRIMARY OWNED  BACKUP OWNED  REQ AVG MS  TASK AVG MS
      1        1     1        0.00%            1  2,147,483,647           0             0             85            86      6.0946       0.0000
      2        1     1        0.00%            1  2,147,483,647           0             0             86            86      9.2803       0.0000
      3        1     1        0.00%            1  2,147,483,647           0             0             86            85      9.7037       0.0000</markup>

</div>

<h4 id="get-service-distributions">Get Service Distributions</h4>
<div class="section">
<p>The 'get service-distributions' command displays partition distributions for a service.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get service-distributions service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for service-distributions</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl get service-distributions PartitionedCache -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Partition Distributions Scheduled for Service "PartitionedCache"

Machine localhost
    Member 1:
        - scheduled to receive 1 Backup partitions:
           -- 1 from member 6
    Member 7:
        - scheduled to receive 16 Primary partitions:
           -- 16 from member 3
        - scheduled to receive 35 Backup partitions:
           -- 15 from member 1
           -- 16 from member 3
           -- 4 from member 6
    Member 6:
        - scheduled to receive 34 Primary partitions:
           -- 18 from member 1
           -- 16 from member 3
        - scheduled to receive 27 Backup partitions:
           -- 7 from member 3
           -- 20 from member 7</markup>

</div>

<h4 id="get-service-description">Get Service Description</h4>
<div class="section">
<p>The 'get service-description' command displays information regarding a service and it&#8217;s members.
Only available in most recent Coherence versions.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get service-description service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for service-description</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl get service-description PartitionedCache -c local</markup>

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
  -y, --yes           automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl start service PartitionedCache -n 1 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to perform start for service PartitionedCache on node 1? (y/n) y
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
  -y, --yes           automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl stop service PartitionedCache -n 1 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to perform stop for service PartitionedCache on node 1? (y/n) y
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
  -y, --yes           automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl shutdown service PartitionedCache -n 1 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to perform shutdown for service PartitionedCache on node 1? (y/n) y
operation completed</markup>

</div>

<h4 id="set-service">Set Service</h4>
<div class="section">
<p>The 'set service' command sets an attribute for a service across one or member nodes.
The following attribute names are allowed: threadCount, threadCountMin, threadCountMax or
taskHungThresholdMillis or requestTimeoutMillis.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl set service service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -a, --attribute string   attribute name to set
  -h, --help               help for service
  -n, --node string        comma separated node ids to target (default "all")
  -v, --value string       attribute value to set
  -y, --yes                automatically confirm the operation</pre>
</div>

<p>See the <a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/manage/oracle-coherence-mbeans-reference.html">Service MBean Reference</a>
for more information on the above attributes.</p>

<p><strong>Examples</strong></p>

<p>Set the thread count min for a service.</p>

<p>List the services and inspect the current thread count min.</p>

<markup
lang="bash"

>cohctl get services -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>SERVICE NAME         TYPE              MEMBERS  STATUS HA  STORAGE  SENIOR PARTITIONS
Proxy                Proxy                   2  n/a             -1       1         -1
PartitionedTopic     DistributedCache        2  NODE-SAFE        2       2        257
PartitionedCache2    DistributedCache        2  NODE-SAFE        2       2        257
PartitionedCache     DistributedCache        2  NODE-SAFE        2       1        257
ManagementHttpProxy  Proxy                   1  n/a             -1       1         -1</markup>

<markup
lang="bash"

>cohctl get services -o  jsonpath="$.items[?(@.name == 'PartitionedCache')]..['nodeId','name','threadCountMin']"</markup>

<p>Output:</p>

<markup
lang="bash"

>["2","PartitionedCache",1,"1","PartitionedCache",1]</markup>

<div class="admonition note">
<p class="admonition-inline">The above shows that the <code>threadCountMin</code> is 1 for both nodes.</p>
</div>
<p>Set the <code>threadCountMin</code> to 10 for each service member.</p>

<markup
lang="bash"

>cohctl set service PartitionedCache -a threadCountMin -v 10 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Selected service: PartitionedCache
Are you sure you want to set the value of attribute threadCountMin to 10 for all 2 nodes? (y/n) y
operation completed</markup>

<markup
lang="bash"

>cohctl get services -o  jsonpath="$.items[?(@.name == 'PartitionedCache')]..['nodeId','name','threadCountMin']"</markup>

<p>Output:</p>

<markup
lang="bash"

>["2","PartitionedCache",10,"1","PartitionedCache",10]</markup>

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
