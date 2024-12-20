<doc-view>

<h2 id="_federation">Federation</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>There are various commands that allow you to work with and issue Federation commands.</p>

<ul class="ulist">
<li>
<p><router-link to="#get-federation" @click.native="this.scrollFix('#get-federation')"><code>cohctl get federation</code></router-link> - displays federation details for a cluster</p>

</li>
<li>
<p><router-link to="#set-federation" @click.native="this.scrollFix('#set-federation')"><code>cohctl set federation</code></router-link> - sets an attribute for a federated service</p>

</li>
<li>
<p><router-link to="#describe-federation" @click.native="this.scrollFix('#describe-federation')"><code>cohctl describe federation</code></router-link> - displays federation details for a given service and participant</p>

</li>
<li>
<p><router-link to="#get-federation-incoming" @click.native="this.scrollFix('#get-federation-incoming')"><code>cohctl get federation-incoming</code></router-link> - displays incoming federation connections information for a given service and participant</p>

</li>
<li>
<p><router-link to="#start-federation" @click.native="this.scrollFix('#start-federation')"><code>cohctl start federation</code></router-link> - starts federation for a service</p>

</li>
<li>
<p><router-link to="#stop-federation" @click.native="this.scrollFix('#stop-federation')"><code>cohctl stop federation</code></router-link> - stops federation for a service</p>

</li>
<li>
<p><router-link to="#pause-federation" @click.native="this.scrollFix('#pause-federation')"><code>cohctl pause federation</code></router-link> - pauses federation for a service</p>

</li>
<li>
<p><router-link to="#replicate-all" @click.native="this.scrollFix('#replicate-all')"><code>cohctl replicate all</code></router-link> - initiates a replicate of all cache entries for a federated service</p>

</li>
</ul>
<div class="admonition note">
<p class="admonition-inline">This is a Coherence Grid Edition feature only and is not available with Community Edition.</p>
</div>
<p>See the <a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/administer/federating-caches-clusters.html">Coherence Documentation</a> for
more information on Federation.</p>


<h4 id="get-federation">Get Federation</h4>
<div class="section">
<p>The 'get federation' command displays the federation details for a cluster.
You must specify either outgoing, incoming or all to show both. You
can also specify '-o wide' to display addition information.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get federation {outgoing|incoming|all} [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for federation</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display all destinations.</p>

<div class="admonition note">
<p class="admonition-inline">Destinations are clusters that this cluster is sending data to.</p>
</div>
<markup
lang="bash"

>cohctl get federation destinations -c local -m</markup>

<p>Output:</p>

<markup
lang="bash"

>SERVICE         DESTINATION        MEMBERS  STATES    DATA SENT  MSG SENT  REC SENT  CURR AVG BWIDTH
FederatedCache  secondary-cluster        2  [PAUSED]       0 MB         0         0          0.0Mbps</markup>

<p>Display all destinations in wide format.</p>

<markup
lang="bash"

>cohctl get federation destinations -c local -o wide -m</markup>

<p>Output:</p>

<markup
lang="bash"

>SERVICE         DESTINATION        MEMBERS  STATES  DATA SENT  MSG SENT  REC SENT  CURR AVG BWIDTH  AVG APPLY  AVG ROUND TRIP  AVG BACKLOG DELAY  REPLICATE  PARTITIONS  ERRORS  UNACKED
FederatedCache  secondary-cluster        2  [IDLE]     204 MB     1,028     3,348          0.0Mbps      338ms         1,393ms           37,770ms    100.00%          31       0        0</markup>

<p>Using the wide option, the following fields are available in regard to the current (or latest) replicate all operation:</p>

<ol style="margin-left: 15px;">
<li>
REPLICATE - the percent complete for the request

</li>
<li>
PARTITIONS - the total number of partitions completed for the request

</li>
<li>
ERRORS - the number of partitions with error responses for the request

</li>
<li>
UNACKED - the total number of partitions that have been sent but have not yet been acknowledged for the request

</li>
</ol>
<div class="admonition note">
<p class="admonition-inline">The last three attributes are only available in the latest Commercial patches. Check your release notes.</p>
</div>
<p>Display all origins.</p>

<div class="admonition note">
<p class="admonition-inline">Origins are clusters that this cluster is receiving data from.</p>
</div>
<markup
lang="bash"

>cohctl get federation destinations -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>cohctl get federation origins -c local -m

SERVICE         ORIGIN             REMOTE CONNECTIONS  DATA REC  MSG REC  REC REC
FederatedCache  secondary-cluster                   2     20 MB      755    2,577</markup>

<p>Display all origins in wide format.</p>

<markup
lang="bash"

>cohctl get federation origins -c local -o wide -m</markup>

<p>Output:</p>

<markup
lang="bash"

>SERVICE         ORIGIN             REMOTE CONNECTIONS  DATA REC  MSG REC  REC REC  AVG APPLY  AVG BACKLOG DELAY
FederatedCache  secondary-cluster                   2     20 MB      755    2,577    1,456ms              248ms</markup>

</div>

<h4 id="set-federation">Set Federation</h4>
<div class="section">
<p>The 'set federation' command sets an attribute for a federated service. The
following attribute names are allowed: traceLogging.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl set federation service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -a, --attribute string   attribute name to set
  -h, --help               help for federation
  -v, --value string       attribute value to set
  -y, --yes                automatically confirm the operation</pre>
</div>

<p>Set <code>traceLogging</code> to true for a federated service.</p>

<markup
lang="bash"

>cohctl set federation FederatedPartitionedPofCache -a traceLogging -v true -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to set the value of attribute traceLogging to true for service FederatedPartitionedPofCache? (y/n) y
operation completed</markup>

</div>

<h4 id="describe-federation">Describe Federation</h4>
<div class="section">
<p>The 'describe federation' command displays the federation details for a given
service, type and participant. Specify -T to set type outgoing or incoming and -p for participant.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl describe federation service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help                 help for federation
  -p, --participant string   participant to apply to (default "all")
  -T, --type string          type to describe outgoing or incoming (default "outgoing")
  -v, --verbose              include verbose output including all attributes</pre>
</div>

<p><strong>Examples</strong></p>

<p>Describe federation destination details on a cluster for the participant secondary-cluster.</p>

<div class="admonition note">
<p class="admonition-inline">This indicates to display data being sent <code>to</code> the destination <code>secondary-cluster</code>.</p>
</div>
<markup
lang="bash"

>cohctl describe federation -c local FederatedPartitionedPofCache -p secondary-cluster -T destinations</markup>

<p>Output:</p>

<markup
lang="bash"

>FEDERATION DETAILS
------------------
Service:     FederatedCache
Type:        destinations
Participant: secondary-cluster

NODE ID  STATE  DATA SENT  MSG SENT  REC SENT  CURR BWIDTH
      1  IDLE      43,205        40        40      0.0Mbps
      2  IDLE      43,391        42        42      0.0Mbps</markup>

<p>Describe federation origin details on a cluster for the participant secondary-cluster.</p>

<div class="admonition note">
<p class="admonition-inline">This indicates to display data being sent <code>from</code> the destination <code>secondary-cluster</code>.</p>
</div>
<markup
lang="bash"

>cohctl describe federation -c local FederatedPartitionedPofCache -p secondary-cluster -T origins -o wide</markup>

<p>Output:</p>

<markup
lang="bash"

>FEDERATION DETAILS
------------------
Service:     FederatedPartitionedPofCache
Type:        origins
Participant: secondary-cluster

NODE ID  CONNECTED  DATA SENT  MSG SENT  REC SENT  AVG APPLY  AVG BACKLOG DELAY
      1          1     83,029        67        67       15ms              440ms
      2          1     87,871        68        68       15ms              350ms</markup>

</div>

<h4 id="get-federation-incoming">Get Federation Incoming Connections</h4>
<div class="section">
<p>The 'get federation-incoming' command displays the incoming connections for a given
service and participant. Specify -p for participant.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get federation-incoming service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help                 help for federation-incoming
  -p, --participant string   participant to apply to</pre>
</div>

<p><strong>Examples</strong></p>

<p>Get incoming federation connections for <code>FederatedCache</code> service and incoming participant <code>Sydney</code>:</p>

<markup
lang="bash"

>cohctl get federation-incoming FederatedCache -p Sydney -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>INCOMING FEDERATION CONNECTIONS
------------------------------
Service:     FederatedCache
Type:        origins
Participant: Sydney

NODE ID  TIMESTAMP                ADDRESS          MACHINE ID  LOCATION                         ROLE
      1  2024-10-07 12:51:03.624  127.0.0.1:62049  10131       machine:localhost,process:21341  CoherenceDemoServer
      2  2024-10-07 13:25:02.737  127.0.0.1:63456  10131       machine:localhost,process:23893  CoherenceDemoServer
      3  2024-10-07 13:25:06.468  127.0.0.1:63475  10131       machine:localhost,process:23894  CoherenceDemoServer</markup>

</div>

<h4 id="start-federation">Start Federation</h4>
<div class="section">
<p>The 'start federation' command starts federation on a service. There
are various options available using '-M' including:
- with-sync - start after federating all cache entries
- no-backlog - clear any initial backlog and start federating
You may also specify a participant otherwise the command will apply to all participants.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl start federation service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help                 help for federation
  -p, --participant string   participant to apply to (default "all")
  -M, --start-mode string    the start mode. Leave blank for normal or specify with-sync or no-backlog
  -y, --yes                  automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<p>Start Federation for all participants.</p>

<markup
lang="bash"

>cohctl start federation FederatedCache -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to start federation for service FederatedCache for participants [secondary-cluster] ? (y/n) y
operation completed</markup>

<p>Start Federation for a specific participant.</p>

<markup
lang="bash"

>cohctl start federation FederatedCache -p secondary-cluster -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to start federation for service FederatedCache for participants [secondary-cluster] ? (y/n) y
operation completed</markup>

<p>Start Federation for a specific participant with no backlog.</p>

<markup
lang="bash"

>cohctl start federation FederatedCache -p secondary-cluster -m no-backlog -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to start (no-backlog) federation for service FederatedCache for participants [secondary-cluster] ? (y/n) y
operation completed</markup>

</div>

<h4 id="stop-federation">Stop Federation</h4>
<div class="section">
<p>The 'stop federation' command stops federation on a service. There
You may also specify a participant otherwise the command will apply to all participants.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl stop federation service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help                 help for federation
  -p, --participant string   participant to apply to (default "all")
  -y, --yes                  automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<p>Stop Federation for all participants.</p>

<markup
lang="bash"

>cohctl stop federation FederatedCache -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to stop federation for service FederatedCache for participants [secondary-cluster] ? (y/n) y
operation completed</markup>

<p>Stop Federation for a specific participant.</p>

<markup
lang="bash"

>cohctl stop federation FederatedCache -p secondary-cluster -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to start federation for service FederatedCache for participants [secondary-cluster] ? (y/n) y
operation completed</markup>

</div>

<h4 id="pause-federation">Pause Federation</h4>
<div class="section">
<p>The 'pause' command stops federation on a service.
You may also specify a participant otherwise the command will apply to all participants.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl pause federation service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help                 help for federation
  -p, --participant string   participant to apply to (default "all")
  -y, --yes                  automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<p>Pause Federation for all participants.</p>

<markup
lang="bash"

>cohctl pause FederatedCache -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to pause federation for service FederatedCache for participants [secondary-cluster] ? (y/n) y
operation completed</markup>

</div>

<h4 id="replicate-all">Replicate All</h4>
<div class="section">
<p>The 'replicate all' command replicates all caches for a federated service.
You must specify a participant to replicate for.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl replicate all service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help                 help for all
  -p, --participant string   participant to apply to
  -y, --yes                  automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<p>Replicate all for a specific participant</p>

<markup
lang="bash"

>cohctl replicate all FederatedCache -p secondary-cluster -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to replicateAll federation for service FederatedCache for participants [secondary-cluster] ? (y/n) y
operation complete</markup>

<div class="admonition note">
<p class="admonition-inline">When this command returns, the replicate all request has been sent to the cluster but may not yet be complete.
You should use the command <code>cohctl get federation destinations -o wide</code> to show the replication percent complete.</p>
</div>
</div>
</div>

<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/administer/federating-caches-clusters.html">Federating Caches in the Coherence Documentation</a></p>

</li>
<li>
<p><router-link to="/docs/reference/20_services">Services</router-link></p>

</li>
</ul>
</div>
</div>
</doc-view>
