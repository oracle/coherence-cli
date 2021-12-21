<doc-view>

<h2 id="_clusters">Clusters</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>There are various cluster commands that allow you to work with and manage cluster connections.</p>

<ul class="ulist">
<li>
<p><router-link to="#add-cluster" @click.native="this.scrollFix('#add-cluster')"><code>cohctl add cluster</code></router-link> - adds a cluster connection</p>

</li>
<li>
<p><router-link to="#discover-clusters" @click.native="this.scrollFix('#discover-clusters')"><code>cohctl discover clusters</code></router-link> - discovers clusters using the Name Service</p>

</li>
<li>
<p><router-link to="#remove-cluster" @click.native="this.scrollFix('#remove-cluster')"><code>cohctl remove cluster</code></router-link> - removes a cluster connection</p>

</li>
<li>
<p><router-link to="#get-clusters" @click.native="this.scrollFix('#get-clusters')"><code>cohctl get clusters</code></router-link> - returns the list of cluster connections</p>

</li>
<li>
<p><router-link to="#describe-cluster" @click.native="this.scrollFix('#describe-cluster')"><code>cohctl describe cluster</code></router-link> - describes a cluster referred to by a cluster connection</p>

</li>
</ul>

<h4 id="add-cluster">Add Cluster</h4>
<div class="section">
<p>The 'add cluster' command adds a new connection to a Coherence cluster. You can
specify the full url such as <a id="" title="" target="_blank" href="https://&lt;host&gt;:&lt;management-port&gt;/management/coherence/cluster">https://&lt;host&gt;:&lt;management-port&gt;/management/coherence/cluster</a>.
You can also specify host and port (for http connections) and the url will be automatically
populated constructed.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl add cluster connection-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help          help for cluster
  -t, --type string   Connection type, http (default "http")
  -u, --url string    Connection URL</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl add cluster local -u http://localhost:30000/management/coherence/cluster
Added cluster local with type http and URL http://localhost:30000/management/coherence/cluster</markup>

<p><strong>Notes</strong></p>

<p>Cluster connections are in one of two formats:</p>

<ul class="ulist">
<li>
<p>Standalone Coherence - <a id="" title="" target="_blank" href="http://host:management-port/management/coherence/cluster">http://host:management-port/management/coherence/cluster</a></p>

</li>
<li>
<p>WebLogic Server - <a id="" title="" target="_blank" href="http://&lt;admin-host&gt;:&lt;admin-port&gt;/management/coherence/latest/clusters">http://&lt;admin-host&gt;:&lt;admin-port&gt;/management/coherence/latest/clusters</a></p>

</li>
</ul>
<p>If you are connecting to WebLogic Server or a Management over REST endpoint that has authentication, you can
specify the user using the <code>-U</code> option. To specify a password, you have the following options:</p>

<ul class="ulist">
<li>
<p>Enter the password when prompted for, or</p>

</li>
<li>
<p>Use the <code>-i</code> or <code>--stdin</code> option to read the password from standard in. (Useful for GitHub actions or automated processes)</p>

</li>
</ul>
<p>You can also specify just a host:port and <code>cohctl</code> will construct a http connection using those in the correct
format.</p>

<markup
lang="bash"

>$ cohctl add cluster local -u localhost:30000
Added cluster local with type http and URL http://localhost:30000/management/coherence/cluster</markup>

<div class="admonition note">
<p class="admonition-inline">If you wish to add a <code>https</code> connection, you must enter the entire URL.</p>
</div>
<div class="admonition note">
<p class="admonition-inline">You can set the <code>HTTP_PROXY</code> environment variable to use a Proxy Server to connect to your cluster endpoint.</p>
</div>
</div>

<h4 id="discover-clusters">Discover Clusters</h4>
<div class="section">
<p>The 'discover clusters' command discovers Coherence clusters using the Name Service.
You can specify a list of either host:port pairs or if you specify a host name the default cluster
port of 7574 will be used.
You will be presented with a list of clusters that have Management over REST configured and
you can confirm if you wish to add the discovered clusters.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl discover clusters [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help            help for clusters
  -I, --ignore          Ignore errors from NS lookup
  -t, --timeout int32   Timeout in seconds for NS Lookup requests (default 30)
  -y, --yes             Automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl discover clusters
Attempting to discover clusters using the following NameService addresses: [localhost]
Discovering Management URL for my-cluster on localhost:7574 ...
Discovering Management URL for test-cluster on localhost:63868 ...

Clusters found:    2
Without Http Mgmt: 1
With Http Mgmt:    1

The following clusters do not have Management over REST enabled and cannot be added
  Cluster: test-cluster, Name Service address: localhost:63868

CONNECTION  CLUSTER NAME  HOST       NS PORT  URL
my-cluster  my-cluster    localhost     7574  http://127.0.0.1:30000/management/coherence/cluster

Are you sure you want to add the above 1 cluster(s)? (y/n) y
Added cluster my-cluster with type http and URL http://127.0.0.1:30000/management/coherence/cluster

coherence-cli$ (master)$ cohctl get clusters
CONNECTION  TYPE  URL                                                  VERSION     CLUSTER NAME  TYPE        CTX
my-cluster  http  http://127.0.0.1:30000/management/coherence/cluster  14.1.2.0.0  my-cluster    Standalone</markup>

<div class="admonition note">
<p class="admonition-inline">The cluster connection is automatically generated from the cluster name. If it already exists you will
be asked for specify a name.</p>
</div>
<div class="admonition note">
<p class="admonition-inline">If there are two or more Management URL&#8217;s, you will be asked to select one.</p>
</div>
</div>

<h4 id="remove-cluster">Remove Cluster</h4>
<div class="section">
<p>The 'remove cluster' command removes a cluster connection.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl remove cluster connection-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for cluster</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl remove cluster local
Removed connection for cluster local</markup>

<div class="admonition note">
<p class="admonition-inline">This command only removes the connection to the cluster that <code>cohctl</code> stores. It does not
affect the running Coherence cluster in any way.</p>
</div>
</div>

<h4 id="get-clusters">Get Clusters</h4>
<div class="section">
<p>The 'get clusters' command displays the list of cluster connections.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get clusters [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for clusters</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl get clusters

CONNECTION  TYPE  URL                                                  VERSION  CLUSTER NAME  CLUSTER TYPE  CTX
local       http  http://localhost:30000/management/coherence/cluster  21.06.1  my-cluster    Standalone</markup>

<div class="admonition note">
<p class="admonition-inline">An asterix will show in the <code>CTX</code> column if the cluster has been set using the <code>cohctl set context</code> command.</p>
</div>
</div>

<h4 id="describe-cluster">Describe Cluster</h4>
<div class="section">
<p>The 'describe cluster' command shows cluster information related to a specific
cluster connection, including: cluster overview, members, machines, services, caches,
reporters, proxy servers and Http servers. You can specify '-o wide' to display
addition information.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl describe cluster cluster-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help      help for cluster
  -v, --verbose   Include verbose output including individual members, reporters and executor details</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl describe cluster local</markup>

</div>
</div>

<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.0/rest-reference/quick-start.html">Setting up Management over REST</a></p>

</li>
<li>
<p><router-link to="/docs/reference/10_contexts">Contexts</router-link></p>

</li>
<li>
<p><router-link to="/docs/reference/45_nslookup">NS Lookup</router-link></p>

</li>
</ul>
</div>
</div>
</doc-view>
