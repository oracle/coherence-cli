<doc-view>

<h2 id="_ns_lookup">NS Lookup</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>The CLI provides an implementation of the Name Service lookup to query a clusters Name Service
endpoint for various query values.</p>

<p>You can provide zero or more host/port pairs to this command. If you do not provide a host, <code>localhost</code> will
be used and if you do not provide a port, then the default port <code>7574</code> will be used.</p>

<ul class="ulist">
<li>
<p><router-link to="#nslookup" @click.native="this.scrollFix('#nslookup')"><code>cohctl nslookup</code></router-link> - displays persistence information for a cluster</p>

</li>
</ul>

<h4 id="nslookup">NS Lookup</h4>
<div class="section">
<p>The 'nslookup' command looks up various Name Service endpoints for a cluster host/port.
The various options to pass via -q option include: Cluster/name, Cluster/info, NameService/string/Cluster/foreign,
NameService/string/management/HTTPManagementURL, NameService/string/management/JMXServiceURL, and
NameService/string/metrics/HTTPMetricsURL,
NameService/string/Cluster/foreign/&lt;clustername&gt;/NameService/localPort</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl nslookup &lt;host:port&gt; [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help            help for nslookup
  -I, --ignore          Ignore errors from NS lookup
  -q, --query string    Query string to pass to Name Service lookup
  -t, --timeout int32   Timeout in seconds for NS Lookup requests (default 30)</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display the name of the cluster on localhost port 7574.</p>

<markup
lang="bash"

>$ cohctl nslookup -q Cluster/name
cluster1</markup>

<p>Display the cluster information for the cluster on localhost:7574.</p>

<markup
lang="bash"

>$ cohctl nslookup -q Cluster/info localhost:7574
Name=cluster1, ClusterPort=7574

WellKnownAddressList(
  172.18.0.2
  )

MasterMemberSet(
  ThisMember=Member(Id=1, Timestamp=2021-11-05 00:15:21.501, Address=172.18.0.2:37697, MachineId=47438, Location=site:Site1,machine:server1,process:1,member:member1, Role=OracleCoherenceCliTestingRestServer)
  OldestMember=Member(Id=1, Timestamp=2021-11-05 00:15:21.501, Address=172.18.0.2:37697, MachineId=47438, Location=site:Site1,machine:server1,process:1,member:member1, Role=OracleCoherenceCliTestingRestServer)
  ActualMemberSet=MemberSet(Size=2
    Member(Id=1, Timestamp=2021-11-05 00:15:21.501, Address=172.18.0.2:37697, MachineId=47438, Location=site:Site1,machine:server1,process:1,member:member1, Role=OracleCoherenceCliTestingRestServer)
    Member(Id=2, Timestamp=2021-11-05 00:15:24.98, Address=172.18.0.3:42019, MachineId=47439, Location=site:Site1,machine:server2,process:1,member:member2, Role=OracleCoherenceCliTestingRestServer)
    )
  MemberId|ServiceJoined|MemberState|Version|Edition
    1|2021-11-05 00:15:21.501|JOINED|14.1.1.0.6|GE,
    2|2021-11-05 00:15:24.98|JOINED|14.1.1.0.6|GE
  RecycleMillis=1200000
  RecycleSet=MemberSet(Size=0
    )
  )

TcpRing{Connections=[2]}
IpMonitor{Addresses=1, Timeout=15s}</markup>

<p>Display the local cluster and foreign clusters registered with the Name Service on localhost:7574.</p>

<markup
lang="bash"

>$ cohctl nslookup -q Cluster/name
cluster1

$ cohctl nslookup -q NameService/string/Cluster/foreign
[cluster3, cluster2]</markup>

<p>Display the Management over REST endpoint for the local cluster.</p>

<markup
lang="bash"

>$ cohctl nslookup -q NameService/string/management/HTTPManagementURL
[http://127.0.0.1:51078/management/coherence/cluster]</markup>

<p>Display the local Name Serivce port for a foreign registered cluster.</p>

<markup
lang="bash"

>$ cohctl nslookup -q NameService/string/Cluster/foreign/cluster2/NameService/localPort
51065</markup>

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
