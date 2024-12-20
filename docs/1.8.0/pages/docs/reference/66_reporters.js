<doc-view>

<h2 id="_reporters">Reporters</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>There are various commands that allow you to work with and manage Reporters.</p>

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
<li>
<p><router-link to="#set-reporter" @click.native="this.scrollFix('#set-reporter')"><code>cohctl set reporter</code></router-link> - sets a reporter attribute for one or more members</p>

</li>
<li>
<p><router-link to="#run-report" @click.native="this.scrollFix('#run-report')"><code>cohctl run report</code></router-link> - runs a report on a specific node and returns the report output in JSON</p>

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

>cohctl get reporters -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>NODE ID  STATE    CONFIG FILE               OUTPUT PATH      BATCH#  LAST REPORT  LAST RUN   AVG RUN  INTERVAL  AUTOSTART
      1  Stopped  reports/report-group.xml  /u01/reports/.        0                    0ms  0.0000ms        60  false
      2  Stopped  reports/report-group.xml  /u01/reports/.        0                    0ms  0.0000ms        60  false
      3  Stopped  reports/report-group.xml  /u01/reports/.        0                    0ms  0.0000ms        60  false
      4  Stopped  reports/report-group.xml  /u01/reports/.        0                    0ms  0.0000ms        60  false</markup>

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

>cohctl describe reporter 1 -c local</markup>

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
<pre>  -h, --help   help for reporter
  -y, --yes    automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl start reporter 1 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to start the reporter on node 1? (y/n) y
Reporter has been started on node 1</markup>

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
<pre>  -h, --help   help for reporter
  -y, --yes    automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl stop reporter 1 -c local -y</markup>

<p>Output:</p>

<markup
lang="bash"

>Reporter has been stopped on node 1</markup>

</div>

<h4 id="set-reporter">Set Reporter</h4>
<div class="section">
<p>The 'set reporter' command sets an attribute for one or more reporter nodes.
You can specify 'all' to change the value for all nodes, or specify a comma separated
list of node ids. The following attribute names are allowed:
configFile, currentBatch, intervalSeconds or outputPath.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl set reporter {node-ids|all} [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -a, --attribute string   attribute name to set
  -h, --help               help for reporter
  -v, --value string       attribute value to set
  -y, --yes                automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<p>Set the output path for all reporters to <code>/reports</code>.</p>

<markup
lang="bash"

>cohctl get reporters -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>NODE ID  STATE    CONFIG FILE               OUTPUT PATH  BATCH#  LAST REPORT  LAST RUN   AVG RUN  INTERVAL  AUTOSTART
      1  Stopped  reports/report-group.xml  /u01              0                    0ms  0.0000ms        60  false
      2  Stopped  reports/report-group.xml  /u01              0                    0ms  0.0000ms        60  false</markup>

<p>Set path.</p>

<markup
lang="bash"

>cohctl set reporter all -a outputPath -v /tmp -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to set the value of attribute outputPath to /tmp for all 2 reporter nodes? (y/n) y
operation completed</markup>

<p>Display the reporters.</p>

<markup
lang="bash"

>cohctl get reporters -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>NODE ID  STATE    CONFIG FILE               OUTPUT PATH  BATCH#  LAST REPORT  LAST RUN   AVG RUN  INTERVAL  AUTOSTART
      1  Stopped  reports/report-group.xml  /tmp              0                    0ms  0.0000ms        60  false
      2  Stopped  reports/report-group.xml  /tmp              0                    0ms  0.0000ms        60  false</markup>

<p>Set the interval for reporter on node 1 to 120 seconds.</p>

<markup
lang="bash"

>cohctl get reporters -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>NODE ID  STATE    CONFIG FILE               OUTPUT PATH  BATCH#  LAST REPORT  LAST RUN   AVG RUN  INTERVAL  AUTOSTART
      1  Stopped  reports/report-group.xml  /tmp              0                    0ms  0.0000ms        60  false
      2  Stopped  reports/report-group.xml  /tmp              0                    0ms  0.0000ms        60  false

ohctl set reporter 1 -a intervalSeconds -v 120 -c local

Are you sure you want to set the value of attribute intervalSeconds to 120 for 1 node(s)? (y/n) y
operation completed</markup>

<p>Display the reporters</p>

<markup
lang="bash"

>cohctl get reporters -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>NODE ID  STATE    CONFIG FILE               OUTPUT PATH  BATCH#  LAST REPORT  LAST RUN   AVG RUN  INTERVAL  AUTOSTART
      1  Stopped  reports/report-group.xml  /tmp              0                    0ms  0.0000ms       120  false
      2  Stopped  reports/report-group.xml  /tmp              0                    0ms  0.0000ms        60  false</markup>

</div>

<h4 id="run-report">Run</h4>
<div class="section">
<p>The 'run report' command runs a report on a specific node and returns the report output in JSON.
The report name should not include the .xml extension and will have the 'report' prefix added. E.g.
'report-node' will expand to 'reports/report-node.xml'. A HTTP 400 will be returned if the report name is not valid.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl run report report-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help       help for report
  -n, --node int   node to run report on</pre>
</div>

<div class="admonition note">
<p class="admonition-inline">The otuput will always be JSON. You can use <code>-o jsonpath=&#8230;&#8203;</code> to use jsonpath expression or pipe through to a utility such as <code>jq</code>.</p>
</div>
<p>This REST endpoint that this command uses is only available in the most recent Coherence releases.
You will receive a HTTP 400 error if it is not supported in your Coherence version.</p>

<markup
lang="bash"

>cohctl run report report-node -c local -n 1</markup>

<p>Output:</p>

<markup
lang="bash"

>{"items":[{"RefreshTime":"Tue Oct 15 09:07:55 AWST 2024","ReportTime":"Tue Oct 15 09:07:55 AWST 2024",
...
"RoleName":"CoherenceServer","Addres,"BatchCounter":"0","rowID":3}]}</markup>

<div class="admonition note">
<p class="admonition-inline">The output above is truncated for readability.</p>
</div>
</div>
</div>

<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/reference/20_services">Services</router-link></p>

</li>
<li>
<p><router-link to="/docs/examples/10_jsonpath">Using JSONPath</router-link></p>

</li>
</ul>
</div>
</div>
</doc-view>
