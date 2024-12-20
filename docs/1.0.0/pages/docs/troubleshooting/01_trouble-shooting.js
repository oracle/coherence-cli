<doc-view>

<h2 id="_troubleshooting_guide">Troubleshooting Guide</h2>
<div class="section">
<p>The purpose of this page is to list troubleshooting guides and work-arounds for issues that you may run into when using the Coherence CLI.
This page will be updated and maintained over time to include common issues we see from customers.</p>

</div>

<h2 id="_contents">Contents</h2>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="#bad" @click.native="this.scrollFix('#bad')">Bad CPU type in executable on macOS</router-link></p>

</li>
<li>
<p><router-link to="#execute" @click.native="this.scrollFix('#execute')">Cannot execute [Exec format error] on Linux</router-link></p>

</li>
<li>
<p><router-link to="#jfr" @click.native="this.scrollFix('#jfr')">Null returned in JFR commands</router-link></p>

</li>
<li>
<p><router-link to="#services" @click.native="this.scrollFix('#services')">Cannot find services with quotes in their names</router-link></p>

</li>
</ul>

<h3 id="bad">Bad CPU type in executable on macOS</h3>
<div class="section">

<h4 id="_problem">Problem</h4>
<div class="section">
<p>You receive a message similar to the following when trying to run the CLI on macOS:</p>

<markup
lang="command"

>/usr/local/bin/cohctl: Bad CPU type in executable</markup>

</div>

<h4 id="_solution">Solution</h4>
<div class="section">
<p>This is most likely caused by installing the incorrect macOS .pkg for your architecture.  E.g. you may have an AMD Mac and trying to install the
Apple Silicon version or visa-versa.</p>

<p>Refer to the <router-link to="/docs/installation/01_installation">Coherence CLI Installation section</router-link> to uninstall
<code>cohctl</code> and download the correct pkg for your architecture.</p>

<p>You can run the <code>uname -a</code> command from a terminal and the output will indicate which type of architecture you have. The last value on the line it will be either <code>x86_64</code> for AMD or <code>arm64</code> for M1.</p>

<p><strong>AMD Processor</strong></p>

<markup
lang="command"

>$ uname -a
Darwin ... RELEASE_X86_64 x86_64</markup>

<p><strong>Apple Silicon (M1) Processor</strong></p>

<markup
lang="command"

>$ uname -a
Darwin ... RELEASE_ARM64_T8101 arm64</markup>

<div class="admonition note">
<p class="admonition-inline">Output above has been truncated for brevity.</p>
</div>
</div>
</div>

<h3 id="execute">Cannot execute [Exec format error] on Linux</h3>
<div class="section">

<h4 id="_problem_2">Problem</h4>
<div class="section">
<p>You receive a message similar to the following when trying to run the CLI on Linux:</p>

<markup
lang="command"

>cohctl: cannot execute [Exec format error]</markup>

</div>

<h4 id="_solution_2">Solution</h4>
<div class="section">
<p>This is most likely caused by installing the incorrect linux executable for your architecture.  E.g. you may have an AMD Linux machine and trying to use
the ARM version or visa-versa.</p>

<p>Refer to the <router-link to="/docs/installation/01_installation">Coherence CLI Installation section</router-link> to uninstall
<code>cohctl</code> and download the correct binary for your architecture.</p>

</div>
</div>

<h3 id="jfr">Null returned in JFR commands</h3>
<div class="section">

<h4 id="_problem_3">Problem</h4>
<div class="section">
<p>You see something similar to the following when running Java Flight Recorder (JFR) commands, where there is a null
instead of the member number.</p>

<markup
lang="bash"

>$ cohctl get jfrs -c local

null-&gt;
   No available recordings.
   Use JFR.start to start a recording.
null-&gt;
   No available recordings.
   Use JFR.start to start a recording.</markup>

</div>

<h4 id="_solution_3">Solution</h4>
<div class="section">
<p>Then this is a known issue. To resolve you should apply the most recent available
Coherence patch on version you are using to resolve this.</p>

</div>
</div>

<h3 id="services">Cannot find services with quotes in their names</h3>
<div class="section">

<h4 id="_problem_4">Problem</h4>
<div class="section">
<p>You are unable to describe or query services with quotes in their names.</p>

<p>Some Coherence services may have quotes in their names, especially if they contain a scope which is
delimited by a colon, as in WebLogic Server.
In these cases when you want to specify a service name you must enclose the whole service name in single quotes.</p>

<p>For example, take a look at the services for this WebLogic Server instance:</p>

<markup
lang="bash"

>$ cohctl get services -c wls -U weblogic
Enter password: ********

SERVICE NAME                      TYPE              MEMBERS  STATUS HA  STORAGE  PARTITIONS
"ExampleGAR:PartitionedPofCache"  DistributedCache        4  NODE-SAFE        2         257</markup>

<p>If we issue the following command you will see the error below.</p>

<markup
lang="bash"

>$ cohctl describe service "ExampleGAR:PartitionedPofCache" -c wls -U weblogic
Enter password: *******

Error: unable to find service with service name 'ExampleGAR:PartitionedPofCache'</markup>

</div>

<h4 id="_solution_4">Solution</h4>
<div class="section">
<p>You must surround any service names that have double quotes with single quotes.</p>

<markup
lang="bash"

>cohctl describe service '"ExampleGAR:PartitionedPofCache"' -c wls -U weblogic
Enter password: ******

SERVICE DETAILS
---------------
Name                                :  "ExampleGAR:PartitionedPofCache"
Type                                :  [DistributedCache]
Backup Count                        :  [1]
Backup Count After Writebehind      :  [1]
....</markup>

</div>
</div>
</div>
</doc-view>