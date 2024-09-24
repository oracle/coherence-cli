<doc-view>

<h2 id="_securing_cli_access">Securing CLI Access</h2>
<div class="section">
<p>The Coherence CLI accesses cluster information using the Management over REST endpoint for the cluster as described in the
<a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/rest-reference/quick-start.html">Coherence documentation</a>.</p>

<p>Coherence HTTP Management server authentication and authorization are disabled by
default. We recommend that this is enabled as outlined in the sections below.</p>

<p>Another option for securing access to the management endpoint is to restrict HTTP access to the REST endpoint from trusted or management subnets
using standard networking firewall rules.</p>


<h3 id="_enabling_basic_authentication">Enabling Basic Authentication</h3>
<div class="section">
<p>To enable basic authentication for Management over REST, please follow the instructions in the
<a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/secure/securing-oracle-oracle-http-management-rest-server.html#GUID-816E45C4-2F52-4576-BC09-CF0B6E873CBA">basic authentication</a> section
of the Coherence documentation.</p>

</div>

<h3 id="_enabling_tls_for_management_over_rest_access">Enabling TLS For Management over REST Access</h3>
<div class="section">
<p>To enable TLS to provide authentication for Management over REST, please follow the instructions in the
<a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/secure/securing-oracle-oracle-http-management-rest-server.html#GUID-7FC70A65-FC2F-4CBE-8F7D-7CBC3CDAA10A">SSL authentication</a>
section of the Coherence documentation.</p>

</div>

<h3 id="_connecting_to_a_tls_enabled_cluster">Connecting to a TLS enabled cluster</h3>
<div class="section">
<p>Once you have enabled TLS you can configure the following environment variables if you need to add client certificates or additional trust stores.</p>

<markup
lang="bash"

>export COHERENCE_TLS_CLIENT_CERT=/path/to/client/certificate
export COHERENCE_TLS_CLIENT_KEY=/path/path/to/client/key
export COHERENCE_TLS_CERTS_PATH=/path/to/cert/to/be/added/for/trust</markup>

<p>If you are connecting a cluster with self-signed certificates, you must set the following to ignore invalid certificates:</p>

<markup
lang="bash"

>cohctl set ignore-certs true</markup>

<p>Output:</p>

<markup
lang="bash"

>Value is now set to true</markup>

<div class="admonition note">
<p class="admonition-inline">This is not recommended and should not be used for production systems.</p>
</div>
<p>You can then add your cluster via specifying HTTPS as the protocol:</p>

<markup
lang="bash"

>cohctl add cluster tls -u https://host:30000/management/coherence/cluster</markup>

<p>You will receive the following message every time you run a command if you ignore certificate errors:</p>

<markup
lang="bash"

>WARNING: SSL Certificate validation has been explicitly disabled</markup>

</div>

<h3 id="_working_with_basic_authentication_rest_endpoints">Working with basic authentication REST endpoints</h3>
<div class="section">
<p>If you have enabled basic authentication for your Management over REST endpoint, or you are connecting to a WebLogic Server cluster, you must
provide the <code>-U username</code> option on all <code>cohctl</code> commands.</p>

<p>To specify a password, you have the following options:</p>

<ol style="margin-left: 15px;">
<li>
Enter the password when prompted for, or

</li>
<li>
Use the <code>-i</code> or <code>--stdin</code> option to read the password from standard in. (Useful for GitHub actions or automated processes)

</li>
</ol>
<markup
lang="bash"

>cohctl get members -U username</markup>

<p>Output:</p>

<markup
lang="bash"

>Enter password: *****</markup>

</div>

<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/rest-reference/quick-start.html">REST API for Managing Oracle Coherence</a></p>

</li>
<li>
<p><a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/secure/securing-oracle-oracle-http-management-rest-server.html">Securing Oracle Coherence HTTP Management Over REST Server</a></p>

</li>
</ul>
</div>
</div>
</doc-view>
