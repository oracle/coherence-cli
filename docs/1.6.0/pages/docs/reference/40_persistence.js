<doc-view>

<h2 id="_persistence">Persistence</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>There are various commands that allow you to work with and issue Persistence commands.</p>

<ul class="ulist">
<li>
<p><router-link to="#get-persistence" @click.native="this.scrollFix('#get-persistence')"><code>cohctl get persistence</code></router-link> - displays persistence information for a cluster</p>

</li>
<li>
<p><router-link to="#get-snapshots" @click.native="this.scrollFix('#get-snapshots')"><code>cohctl get snapshots</code></router-link> - shows persistence snapshots for a cluster</p>

</li>
<li>
<p><router-link to="#create-snapshot" @click.native="this.scrollFix('#create-snapshot')"><code>cohctl create snapshot</code></router-link> - create a snapshot for a service</p>

</li>
<li>
<p><router-link to="#recover-snapshot" @click.native="this.scrollFix('#recover-snapshot')"><code>cohctl recover snapshot</code></router-link> - recover a snapshot for a service</p>

</li>
<li>
<p><router-link to="#remove-snapshot" @click.native="this.scrollFix('#remove-snapshot')"><code>cohctl remove snapshot</code></router-link> - remove a snapshot for a service</p>

</li>
<li>
<p><router-link to="#archive-snapshot" @click.native="this.scrollFix('#archive-snapshot')"><code>cohctl archive snapshot</code></router-link> - archive a snapshot for a service</p>

</li>
<li>
<p><router-link to="#retrieve-snapshot" @click.native="this.scrollFix('#retrieve-snapshot')"><code>cohctl retrieve snapshot</code></router-link> - retrieve an archived snapshot for a service</p>

</li>
<li>
<p><router-link to="#suspend-service" @click.native="this.scrollFix('#suspend-service')"><code>cohctl suspend service</code></router-link> - suspends a specific service in all the members of a cluster</p>

</li>
<li>
<p><router-link to="#resume-service" @click.native="this.scrollFix('#resume-service')"><code>cohctl resume service</code></router-link> - resumes a specific service in all the members of a cluster</p>

</li>
</ul>
<p>See the <a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/administer/persisting-caches.html">Coherence Documentation</a> for
more information on Persistence.</p>


<h4 id="get-persistence">Get Persistence</h4>
<div class="section">
<p>The 'get persistence' command displays persistence information for a cluster.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get persistence [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for persistence</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display all persistence services.</p>

<markup
lang="bash"

>cohctl get persistence -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Total Active Space Used: 7MB

SERVICE NAME       STORAGE COUNT  PERSISTENCE MODE  ACTIVE SPACE USED  AVG LATENCY  MAX LATENCY  SNAPSHOTS  STATUS
PartitionedTopic               4  active                      652,342      0.000ms          0ms          0  Idle
PartitionedCache2              4  active                      365,946      0.000ms          0ms          0  Idle
PartitionedCache               4  active                    6,331,471      0.242ms        188ms          2  Idle</markup>

</div>

<h4 id="get-snapshots">Get Snapshots</h4>
<div class="section">
<p>The 'get snapshots' command displays snapshots for a cluster. If
no service name is specified then all services are queried. By default
local snapshots are shown, but you can use the -a option to show archived snapshots.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get snapshots [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -a, --archived         if true, returns archived snapshots, otherwise local snapshots
  -h, --help             help for snapshots
  -s, --service string   Service name</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display snapshots for all services.</p>

<markup
lang="bash"

>cohctl get snapshots -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>SERVICE            SNAPSHOT NAME
PartitionedCache2  snapshot-test
PartitionedCache   snapshot-1
PartitionedCache   snapshot-2</markup>

<p>Display snapshots for a specific service.</p>

<markup
lang="bash"

>cohctl get snapshots -c local -s PartitionedCache</markup>

<p>Output:</p>

<markup
lang="bash"

>SERVICE            SNAPSHOT NAME
PartitionedCache   snapshot-1
PartitionedCache   snapshot-2</markup>

<p>Display <strong>archived</strong> snapshots for all services.</p>

<markup
lang="bash"

>cohctl get snapshots -c local -a</markup>

<p>Output:</p>

<markup
lang="bash"

>SERVICE            ARCHIVED SNAPSHOT NAME
PartitionedCache2  snapshot-test
PartitionedCache   snapshot-1</markup>

<p>Display <strong>archived</strong> snapshots for a specific service.</p>

<markup
lang="bash"

>cohctl get snapshots -c local -s PartitionedCache -a</markup>

<p>Output:</p>

<markup
lang="bash"

>SERVICE           ARCHIVED SNAPSHOT NAME
PartitionedCache  snapshot-1</markup>

</div>

<h4 id="create-snapshot">Create Snapshot</h4>
<div class="section">
<p>The 'create snapshot' command creates a snapshot for a given service. If you
do not specify the -y option you will be prompted to confirm the operation.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl create snapshot snapshot-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for snapshot
  -s, --service string   Service name
  -y, --yes              automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl create snapshot my-snapshot -s PartitionedCache -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to perform create snapshot for snapshot my-snapshot and service PartitionedCache? (y/n) y

Operation create snapshot for snapshot my-snapshot on service PartitionedCache invoked
Please use 'cohctl get persistence' to check for idle status to ensure the operation completed</markup>

<div class="admonition note">
<p class="admonition-inline">This and other commands that create, remove, archive, retrieve or recover snapshots submit this request to
the service to perform the operation only. The return of the command prompt does not mean the operation has been
completed on the service.
You should use <code>cohctl get persistence</code> to ensure the status is Idle and check Coherence log files before continuing.</p>
</div>
</div>

<h4 id="recover-snapshot">Recover Snapshot</h4>
<div class="section">
<p>The 'recover snapshot' command recovers a snapshot for a given service.
WARNING: Issuing this command will destroy all service data and replaced with the
data from the requested snapshot.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl recover snapshot snapshot-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for snapshot
  -s, --service string   Service name
  -y, --yes              automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl recover snapshot my-snapshot -s PartitionedCache -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to perform recover snapshot for snapshot my-snapshot and service PartitionedCache? (y/n) y

Operation recover snapshot for snapshot my-snapshot on service PartitionedCache invoked
Please use 'cohctl get persistence' to check for idle status to ensure the operation completed</markup>

<div class="admonition note">
<p class="admonition-inline">This is a destructive command and will remove all current caches for the specified service and
replace them with the contents of the caches in the snapshot.</p>
</div>
</div>

<h4 id="remove-snapshot">Remove Snapshot</h4>
<div class="section">
<p>The 'remove snapshot' command removes a snapshot for a given service.
By default local snapshots are removed, but you can use the -a option to remove archived snapshots.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl remove snapshot snapshot-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -a, --archived         if true, returns archived snapshots, otherwise local snapshots
  -h, --help             help for snapshot
  -s, --service string   Service name
  -y, --yes              automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<p>Remove a local snapshot.</p>

<markup
lang="bash"

>cohctl recover snapshot my-snapshot -s PartitionedCache -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to perform remove snapshot for snapshot my-snapshot and service PartitionedCache? (y/n) y

Operation remove snapshot for snapshot my-snapshot on service PartitionedCache invoked
Please use 'cohctl get persistence' to check for idle status to ensure the operation completed</markup>

<p>Remove an <strong>archived</strong> snapshot.</p>

<markup
lang="bash"

>cohctl recover snapshot my-snapshot -s PartitionedCache -a -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to perform remove archived snapshot for snapshot my-snapshot and service PartitionedCache? (y/n) y

Operation remove archived snapshot for snapshot my-snapshot on service PartitionedCache invoked
Please use 'cohctl get persistence' to check for idle status to ensure the operation completed</markup>

</div>

<h4 id="archive-snapshot">Archive Snapshot</h4>
<div class="section">
<p>The 'archive snapshot' command archives a snapshot for a given service. You must
have an archiver setup on the service for this to be successful.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl archive snapshot snapshot-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for snapshot
  -s, --service string   Service name
  -y, --yes              automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl archive snapshot my-snapshot -s PartitionedCache -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to perform archive snapshot for snapshot my-snapshot and service PartitionedCache? (y/n) y

Operation archive snapshot for snapshot my-snapshot on service PartitionedCache invoked
Please use 'cohctl get persistence' to check for idle status to ensure the operation completed</markup>

<div class="admonition note">
<p class="admonition-inline">When you issue the archive snapshot command, the snapshots on the separate servers are sent to a
central location. Coherence provides a directory archiver implementation which will store the archive on a shared
filesystem available to all members.  You can also create your own archiver implementations.
See <a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/administer/persisting-caches.html">the Coherence documentation</a>
for more details.</p>
</div>
</div>

<h4 id="retrieve-snapshot">Retrieve Snapshot</h4>
<div class="section">
<p>The 'retrieve snapshot' command retrieves an archived snapshot for a given service.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl retrieve snapshot snapshot-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for snapshot
  -s, --service string   Service name
  -y, --yes              automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl retrieve snapshot my-snapshot -s PartitionedCache -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to perform retrieve snapshot for snapshot my-snapshot and service PartitionedCache? (y/n) y

Operation retrieve snapshot for snapshot my-snapshot on service PartitionedCache invoked
Please use 'cohctl get persistence' to check for idle status to ensure the operation completed</markup>

<div class="admonition note">
<p class="admonition-inline">This operation will retrieve and archived snapshot and distribute it across all available members. Once it has
been retrieved it can be recovered. You must ensure that a snapshot with the same name as the archived snapshot does
not exist before you retrieve it.</p>
</div>
</div>

<h4 id="suspend-service">Suspend Service</h4>
<div class="section">
<p>The 'suspend service' command suspends a specific service in all the members of a cluster.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl suspend service service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for service
  -y, --yes    automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl suspend service PartitionedCache -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to perform suspend service for service PartitionedCache? (y/n) y
operation completed</markup>

<div class="admonition note">
<p class="admonition-inline">You can use the command <code>cohctl get services -o wide</code> to show if services have been suspended.</p>
</div>
</div>

<h4 id="resume-service">Resume Service</h4>
<div class="section">
<p>The 'resume service' command resumes a specific service in all the members of a cluster.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl resume service service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for service
  -y, --yes    automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl resume service PartitionedCache -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to perform resume service for service PartitionedCache? (y/n) y
operation completed</markup>

</div>
</div>

<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/administer/persisting-caches.html">Persisting Caches in the Coherence Documentation</a></p>

</li>
<li>
<p><router-link to="/docs/reference/20_services">Services</router-link></p>

</li>
</ul>
</div>
</div>
</doc-view>
