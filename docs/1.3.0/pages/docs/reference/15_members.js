<doc-view>

<h2 id="_members">Members</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>There are various cluster commands that allow you to work with and manage cluster members.</p>

<ul class="ulist">
<li>
<p><router-link to="#get-members" @click.native="this.scrollFix('#get-members')"><code>cohctl get members</code></router-link> - displays the members for a cluster</p>

</li>
<li>
<p><router-link to="#describe-member" @click.native="this.scrollFix('#describe-member')"><code>cohctl describe member</code></router-link> - shows information related to a specific member</p>

</li>
<li>
<p><router-link to="#set-member" @click.native="this.scrollFix('#set-member')"><code>cohctl set member</code></router-link> - sets a member attribute for one or more members</p>

</li>
<li>
<p><router-link to="#shutdown-member" @click.native="this.scrollFix('#shutdown-member')"><code>cohctl shutdown member</code></router-link> - shuts down a members services in a controlled manner</p>

</li>
</ul>

<h4 id="get-members">Get Members</h4>
<div class="section">
<p>The 'get members' command displays the members for a cluster. You
can specify '-o wide' to display addition information.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get members [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help          help for members
  -r, --role string   role name to display (default "all")</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display all members and specify to display memory sizes in MB using <code>-m</code> option.</p>

<markup
lang="bash"

>$ cohctl get members -c local -m

Total cluster members: 3
Cluster Heap - Total: 4,352 MB Used: 250 MB Available: 4,102 MB (94.3%)
Storage Heap - Total: 4,096 MB Used: 201 MB Available: 3,895 MB (95.1%)

NODE ID  ADDRESS         PORT   PROCESS  MEMBER  ROLE                  STORAGE  MAX HEAP  USED HEAP  AVAIL HEAP
      1  192.168.1.117  63984    35372  n/a     Management            true     2,048 MB      91 MB    1,957 MB
      2  192.168.1.117  63995    35398  n/a     TangosolNetCoherence  true     2,048 MB     110 MB    1,938 MB
      3  192.168.1.117  64013    35430  n/a     CoherenceConsole      false      256 MB      49 MB      207 MB</markup>

<div class="admonition note">
<p class="admonition-inline">The default memory display format is bytes but can be changed by using <code>-k</code>, <code>-m</code> or <code>-g</code>.</p>
</div>
<p>Display all members with the role <code>CoherenceConsole</code>.</p>

<markup
lang="bash"

>$ cohctl get members -c local -r CoherenceConsole -m

Total cluster members: 1
Cluster Heap - Total: 256 MB Used: 50 MB Available: 206 MB (80.5%)
Storage Heap - Total: 0 MB Used: 0 MB Available: 0 MB ( 0.0%)

NODE ID  ADDRESS         PORT   PROCESS  MEMBER  ROLE              STORAGE  MAX HEAP  USED HEAP  AVAIL HEAP
      3  192.168.1.117  64013    35430  n/a      CoherenceConsole  false      256 MB      50 MB      206 MB</markup>

<div class="admonition note">
<p class="admonition-inline">You can also use <code>-o wide</code> to display more columns.</p>
</div>
</div>

<h4 id="describe-member">Describe Member</h4>
<div class="section">
<p>The 'describe member' command shows information related to a specific member.
To display extended information about a member, the -X option can be specified with a comma
separated list of platform entries to display for. For example:</p>

<pre>cohctl describe member 1 -X g1OldGeneration,g1EdenSpace</pre>
<p>would display information related to G1 old generation and Eden space.</p>

<p>Full list of options are JVM dependant, but can include the full values or part of the following:
  compressedClassSpace, operatingSystem, metaSpace, g1OldGen, g1SurvivorSpace, g1CodeHeapProfiledNMethods,
  g1CodeHeapNonNMethods, g1OldGeneration g1MetaSpaceManager, g1YoungGeneration, g1EdenSpace,
  g1CodeCacheManager, psScavenge, psEdenSpace, psMarkSweep, codeCache, psOldGen, psSurvivorSpace,
  runtime</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl describe member node-id [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -X, --extended string   include extended information such as g1OldGen, etc. See --help (default "none")
  -h, --help              help for member
  -D, --thread-dump       include a thread dump</pre>
</div>

<p><strong>Examples</strong></p>

<p>Describe member and include heap dump.</p>

<markup
lang="bash"

>$ cohctl describe member 1 -D</markup>

<div class="admonition note">
<p class="admonition-inline">When taking one or more thread dumps, if you want full deadlock analysis, you should set the following system property
on your Coherence JVMS <code>-Dcom.oracle.coherence.common.util.Threads.dumpLocks=FULL</code></p>
</div>
<p>Describe member and include extended information on G1 garbage collection.</p>

<markup
lang="bash"

>$ cohctl describe member 1 -Xg1</markup>

</div>

<h4 id="set-member">Set Member</h4>
<div class="section">
<p>The 'set member' command sets an attribute for one or more member nodes.
You can specify 'all' to change the value for all nodes, or specify a comma separated
list of node ids. The following attribute names are allowed:
loggingLevel, resendDelay, sendAckDelay, trafficJamCount, trafficJamDelay, loggingLimit
or loggingFormat.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl set member {node-ids|all} [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -a, --attribute string   attribute name to set
  -h, --help               help for member
  -v, --value string       attribute value to set
  -y, --yes                automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<p>Set the log level for all members.</p>

<markup
lang="bash"

># Check the log level first
$ cohctl get members -o json | jq | grep loggingLevel
      "loggingLevel": 6,
      "loggingLevel": 6,
      "loggingLevel": 6,

$ cohctl set member all -a loggingLevel -v 6 -c local

re you sure you want to set the value of attribute loggingLevel to 6 for all 3 nodes? (y/n) y
operation completed
$ cohctl get members -o json | jq | grep loggingLevel
      "loggingLevel": 6,
      "loggingLevel": 6,
      "loggingLevel": 6,</markup>

<p>Set the log level to 9 for node id 1.</p>

<markup
lang="bash"

>$ cohctl set member 1 -a loggingLevel -v 9 -c local

Are you sure you want to set the value of attribute loggingLevel to 9 for 1 nodes? (y/n) y
operation completed

$ cohctl get members -o json | jq | grep loggingLevel
      "loggingLevel": 9,
      "loggingLevel": 6,
      "loggingLevel": 6,</markup>

</div>

<h4 id="shutdown-member">Shutdown Member</h4>
<div class="section">
<p>The 'shutdown member' command shuts down all the clustered services that are
running on a specific member via a controlled shutdown. If the services were started using
DefaultCacheServer, then they will be restarted.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl shutdown member node-id [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for member
  -y, --yes    automatically confirm the operation</pre>
</div>

<markup
lang="bash"

>$ cohctl shutdown member 1 -c local
Are you sure you want to shutdown member 1? (y/n) y
operation completed</markup>

<div class="admonition note">
<p class="admonition-inline">This command may take a while to complete. Check the server logs for information regarding the
member&#8217;s status.  If the services were started using DefaultCacheServer, then they will be restarted.</p>
</div>
</div>
</div>

<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/reference/85_diagnostics">Diagnostics</router-link></p>

</li>
</ul>
</div>
</div>
</doc-view>
