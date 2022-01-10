<doc-view>

<h2 id="_reporters">Reporters</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>There are various commands that allow you to work with and manage Reporters servers.</p>

<ul class="ulist">
<li>
<p><router-link to="#get-reporters" @click.native="this.scrollFix('#get-reporters')"><code>cohctl get reporters</code></router-link> - displays the reporters for a cluster</p>

</li>
<li>
<p><router-link to="#describe-reporter" @click.native="this.scrollFix('#describe-reporter')"><code>cohctl describe reporter</code></router-link> - shows information related to a specific reporter</p>

</li>
<li>
<p><router-link to="#start-reporter" @click.native="this.scrollFix('#start-reporter')"><code>cohctl start reporter</code></router-link> - starts a reporter on a specific node</p>

</li>
<li>
<p><router-link to="#stop-reporter" @click.native="this.scrollFix('#stop-reporter')"><code>cohctl stop reporter</code></router-link> - stops a reporter on a specific node</p>

</li>
</ul>

<h4 id="get-reporters">Get Reporters</h4>
<div class="section">
<p>The 'get reporters' command displays the reporters for the cluster.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get reporters [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for reporters</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display all http servers.</p>

<markup
lang="bash"

>$ cohctl get reporters -c local
NODE ID  STATE    CONFIG FILE               OUTPUT PATH      BATCH#  LAST REPORT  LAST RUN   AVG RUN  AUTOSTART
      1  Stopped  reports/report-group.xml  /u01/reports/.        0                    0ms  0.0000ms  false
      2  Stopped  reports/report-group.xml  /u01/reports/.        0                    0ms  0.0000ms  false
      3  Stopped  reports/report-group.xml  /u01/reports/.        0                    0ms  0.0000ms  false
      4  Stopped  reports/report-group.xml  /u01/reports/.        0                    0ms  0.0000ms  false</markup>

</div>

<h4 id="describe-reporter">Describe Reporter</h4>
<div class="section">
<p>The 'describe reporter' command shows information related to a particular reporter.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl describe reporter node-id [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for reporter</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl describe reporter 1 -c local</markup>

</div>

<h4 id="start-reporter">Start Reporter</h4>
<div class="section">
<p>The 'start reporter' command starts the Coherence reporter on the specified node.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl start reporter node-id [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for reporter</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl start reporter 1 -c local</markup>

</div>

<h4 id="stop-reporter">Stop Reporter</h4>
<div class="section">
<p>The 'stop reporter' command stops the Coherence reporter on the specified node.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl stop reporter node-id [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for reporter</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl stop reporter 1 -c local</markup>

</div>
</div>

<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/reference/20_services">Services</router-link></p>

</li>
</ul>
</div>
</div>
</doc-view>
