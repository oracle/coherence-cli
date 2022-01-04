<doc-view>

<h2 id="_what_is_the_coherence_cli">What is the Coherence CLI?</h2>
<div class="section">
<p>The Coherence command line interface, <code>cohctl</code>, is a lightweight tool, in the tradition of tools such as kubectl,
which can be scripted or used interactively to manage and monitor Coherence clusters. You can use <code>cohctl</code> to view cluster information
such as services, caches, members, etc, as well as perform various management operations against clusters.</p>

<p>The CLI accesses clusters using the HTTP Management over REST interface and therefore requires this to be enabled on any clusters
you want to monitor or manage. See the <a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.0/rest-reference/quick-start.html">Coherence Documentation</a>
for more information on setting up Management over REST.</p>

<p>The CLI is certified with all Coherence Community Edition (CE) versions as well as Coherence Commercial versions 12.2.1.4, 14.1.1.0 and above.</p>

<div class="admonition note">
<p class="admonition-inline">The CLI does not replace current management and monitoring tools such as the <a id="" title="" target="_blank" href="https://github.com/oracle/coherence-visualvm">Coherence VisualVM Plugin</a>,
<a id="" title="" target="_blank" href="https://docs.oracle.com/cd/E24628_01/install.121/e24215/coherence_getstarted.htm#GSSOA10121">Enterprise Manager</a>, or <a id="" title="" target="_blank" href="https://oracle.github.io/coherence-operator/docs/latest/#/docs/metrics/040_dashboards">Grafana Dashboards</a>, but compliments and
provides a lightweight and scriptable alternative.</p>
</div>
</div>

<h2 id="_why_use_the_coherence_cli">Why use the Coherence CLI?</h2>
<div class="section">
<p>The CLI compliments your existing Coherence management tools and allows you to:</p>

<ol style="margin-left: 15px;">
<li>
Interactively monitor your Coherence clusters from a lightweight terminal-based interface

</li>
<li>
Monitor service "StatusHA" during rolling restarts of Coherence clusters

</li>
<li>
Script Coherence monitoring and incorporate results into other management tooling

</li>
<li>
Output results in various formats including text, JSON and utilize JsonPath to extract attributes of interest

</li>
<li>
Gather information that may be useful for Oracle Support to help diagnose issues

</li>
<li>
Connect to standalone or WebLogic Server based clusters from commercial versions 12.2.1.4 and above as well as all <a id="" title="" target="_blank" href="https://github.com/oracle/coherence">Coherence Community Edition</a> (CE) versions

</li>
<li>
Retrieve thread dumps and Java Flight Recordings across members

</li>
<li>
Make changes to various modifiable JMX attributes on services, caches and members

</li>
</ol>
</div>

<h2 id="_next_steps">Next Steps</h2>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/installation/01_installation">Install the Coherence CLI</router-link></p>

</li>
<li>
<p><router-link to="/docs/about/03_quickstart">Run the Quick Start</router-link></p>

</li>
<li>
<p><router-link to="/docs/reference/01_overview">Explore the Command Reference</router-link></p>

</li>
</ul>
</div>
</doc-view>