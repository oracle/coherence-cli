<doc-view>

<h2 id="_starter_projects">Starter Projects</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>This <strong>experimental</strong> feature allows you to create example Maven projects to show basic integration of Coherence with various frameworks.
Currently, we support the following integrations:</p>

<ol style="margin-left: 15px;">
<li>
Helidon Microprofile - 4.2.1

</li>
<li>
Spring Boot - spring-boot-starter 3.4.5, coherence-spring 4.3.0

</li>
<li>
Micronaut - micronaut-parent: 4.7.3, micronaut-coherence: 5.0.4

</li>
</ol>

<div class="admonition note">
<p class="admonition-inline">The framework versions may be updated from time to time.</p>
</div>

<p>The example projects generated expose a simple REST API where the data is stored in Coherence caches using each of the different frameworks.</p>

<p>These examples are demo projects only and are meant to provide basic and first steps integrating Coherence with
various frameworks. See the <a target="_blank" href="https://coherence.community/examples.html">Coherence Community</a> website for fully fledged examples using Coherence.</p>

<p>You must have Maven 3.8.+ and JDK 21+ on your system in able to build and run the starter projects.</p>

<div class="admonition note">
<p class="admonition-inline">This is an <strong>experimental</strong> feature that may be changed or removed in a future version.</p>
</div>

<ul class="ulist">
<li>
<p><router-link to="#create-starter" @click.native="this.scrollFix('#create-starter')"><code>cohctl create starter</code></router-link> - create a starter project with Coherence and various frameworks</p>

</li>
</ul>


<h4 id="create-starter">Create Starter</h4>
<div class="section">
<p>The 'create starter' command creates a starter Maven project to use Coherence
with various frameworks including Helidon, Spring Boot and Micronaut. A directory
will be created off the current directory with the same name as the project name.
NOTE: This is an experimental feature only and the projects created are not fully
completed applications. They are a demo/example of how to do basic integration with
each of the frameworks.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl create starter project-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -f, --framework string   the framework to create for: helidon, springboot or micronaut
  -h, --help               help for starter
  -y, --yes                automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<p>Create a starter project using <a target="_blank" href="https://helidon.io/">Helidon</a> Microprofile framework.</p>

<markup
lang="bash"

>cohctl create starter helidon-starter -f helidon</markup>

<p>Output:</p>

<markup
lang="text"

>checking for availability of template helidon...

Create Starter Project
Project Name:       helidon-starter
Framework Type:     helidon
Framework Versions: 4.2.1
Project Path        /tmp/starters/helidon-starter

Are you sure you want to create the starter project above? (y/n) y

Your helidon template project has been saved to /tmp/starters/helidon-starter

Please see the file helidon-starter/readme.txt for instructions</markup>

<p>Inspect the readme.txt</p>

<markup
lang="bash"

>cat helidon-starter/readme.txt</markup>

<p>Output:</p>

<markup
lang="text"

>To run the Helidon starter you must have JDK21+ and maven 3.8.5+.
Change to the newly created directory and run the following to build:

    mvn clean install

To run single server:
    java -jar target/helidon.jar

To run additional server:
    java -Dmain.class=com.tangosol.net.Coherence -Dcoherence.management.http=none -Dserver.port=-1 -jar target/helidon.jar

Add a customer:
    curl -X POST -H "Content-Type: application/json" -d '{"id": 1, "name": "Tim", "balance": 1000}' http://localhost:8080/api/customers

Get a customer:
    curl -s http://localhost:8080/api/customers/1

Get all customers:
    curl -s http://localhost:8080/api/customers

Delete a customer:
    curl -X DELETE http://localhost:8080/api/customers/1</markup>

<p>Follow the instructions above to build and run the example.</p>

</div>

</div>


<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><a target="_blank" href="https://helidon.io/">Helidon Framework</a></p>

</li>
<li>
<p><a target="_blank" href="https://github.com/coherence-community/coherence-spring">Coherence Spring Integration</a></p>

</li>
<li>
<p><a target="_blank" href="https://github.com/micronaut-projects/micronaut-coherence/">Coherence Micronaut Integration</a></p>

</li>
</ul>

</div>

</div>

</doc-view>
