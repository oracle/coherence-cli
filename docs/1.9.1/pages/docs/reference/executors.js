<doc-view>

<h2 id="_executors">Executors</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>There are various commands that allow you to work with and manage Executors.</p>

<ul class="ulist">
<li>
<p><router-link to="#get-executors" @click.native="this.scrollFix('#get-executors')"><code>cohctl get executors</code></router-link> - displays the executors for a cluster</p>

</li>
<li>
<p><router-link to="#describe-executor" @click.native="this.scrollFix('#describe-executor')"><code>cohctl describe executor</code></router-link> - shows information related to a specific executor</p>

</li>
<li>
<p><router-link to="#set-executor" @click.native="this.scrollFix('#set-executor')"><code>cohctl set executor</code></router-link> - sets an attribute for a specific executor across all nodes</p>

</li>
</ul>


<h4 id="get-executors">Get Executors</h4>
<div class="section">
<p>The 'get executors' command displays the executors for a cluster.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get executors [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for executors</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl get executors -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Total executors: 3
Running tasks:   0
Completed tasks: 0

NAME         MEMBER COUNT  IN PROGRESS  COMPLETED  REJECTED  DESCRIPTION
executor1               2            0          0         0  FixedThreadPool(ThreadCount=5, ThreadFactory=default)
executor2               2            0          0         0  SingleThreaded(ThreadFactory=default)
UnNamed                 2            0          0         0  None</markup>

</div>


<h4 id="describe-executor">Describe Executor</h4>
<div class="section">
<p>The 'describe executor' command shows information related to a specific executor.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl describe executor executor-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for executor</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl describe executor executor1 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>EXECUTOR DETAILS
----------------
Name                   :  executor1
Member Id              :  1
Description            :  SingleThreaded(ThreadFactory=default)
Id                     :  3af3cb00-b87d-4b89-ae9f-2107743b0741
Location               :  Member(Id=1, Timestamp=2021-12-02 15:16:21.247, Address=192.168.1.120:64409, MachineId=3603, Location=process:35013, Role=Management)
Member Count           :  0
State                  :  RUNNING
Tasks Completed Count  :  0
Tasks In Progress Count:  0
Tasks Rejected Count   :  0
Trace Logging          :  false

Name                   :  executor1
Member Id              :  2
Description            :  SingleThreaded(ThreadFactory=default)
Id                     :  cd7241ce-2a0a-41f4-85cd-538513fba2ac
Location               :  Member(Id=2, Timestamp=2021-12-02 15:28:50.824, Address=192.168.1.120:64911, MachineId=3603, Location=process:37972, Role=TangosolNetCoherence)
Member Count           :  0
State                  :  RUNNING
Tasks Completed Count  :  0
Tasks In Progress Count:  0
Tasks Rejected Count   :  0
Trace Logging          :  false</markup>

</div>


<h4 id="set-executor">Set Executors</h4>
<div class="section">
<p>The 'set executor' command sets an attribute for a specific executor across
all nodes. The following attribute names are allowed: traceLogging.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl set executor executor-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -a, --attribute string   attribute name to set
  -h, --help               help for executor
  -v, --value string       attribute value to set
  -y, --yes                automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<p>Turn on trace logging for the coherence-concurrent-default-executor executor.</p>

<markup
lang="bash"

>cohctl set executor coherence-concurrent-default-executor -a traceLogging -v true -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to set the value of attribute traceLogging to true for coherence-concurrent-default-executor? (y/n) y
operation completed</markup>

</div>

</div>

</div>

</doc-view>
