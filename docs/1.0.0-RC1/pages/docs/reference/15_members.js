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
  -r, --role string   Role name to display (default "all")</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display all members.</p>

<markup
lang="bash"

>$ cohctl get members -c local

Cluster Heap - Total: 2.500GB, Used: 305MB, Available: 2.202GB (88.1%)

NODE ID  ADDRESS          PORT  PROCESS  MEMBER  ROLE               MAX HEAP  USED HEAP  AVAIL HEAP
1        /192.168.1.124  53216    12096  n/a     CoherenceServer     1.000GB       95MB       929MB
2        /192.168.1.124  53215    12094  n/a     Management            512MB      112MB       400MB
3        /192.168.1.124  53220    12246  n/a     CoherenceServer     1.000GB       98MB       926MB</markup>

<p>Display all members with the role <code>CoherenceServer</code>.</p>

<markup
lang="bash"

>$ cohctl get members -c local -r CoherenceServer
Cluster Heap - Total: 2.000GB, Used: 197MB, Available: 1.808GB (90.4%)

NODE ID  ADDRESS          PORT  PROCESS  MEMBER  ROLE             MAX HEAP  USED HEAP  AVAIL HEAP
1        /192.168.1.124  53216    12096  n/a     CoherenceServer   1.000GB       98MB       926MB
3        /192.168.1.124  53220    12246  n/a     CoherenceServer   1.000GB       99MB       925MB</markup>

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
<pre>  -X, --extended string   Include extended information such as g1OldGen, etc. See --help (default "none")
  -h, --help              help for member
  -D, --thread-dump       Include a thread dump</pre>
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
  -y, --yes                Automatically confirm the operation</pre>
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
