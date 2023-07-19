<doc-view>

<h2 id="_command_completion">Command Completion</h2>
<div class="section">
<p>The Coherence CLI allows you to generate a script that will help you work with the tool
by providing command completion.</p>


<h3 id="_generate_the_script">Generate the Script</h3>
<div class="section">
<p>Use the following command to show the help for the <code>completion</code> command</p>

<markup
lang="bash"

>$ cohctl completion

Generate the autocompletion script for cohctl for the specified shell.
See each sub-command's help for details on how to use the generated script.

Usage:
cohctl completion [command]

Available Commands:
bash        generate the autocompletion script for bash
fish        generate the autocompletion script for fish
powershell  generate the autocompletion script for powershell
zsh         generate the autocompletion script for zsh</markup>

<p>Execute the command for your desired shell. In our case it will be <code>bash</code>.</p>

<markup
lang="bash"

>$ cohctl completion bash &gt; ~/cohctl-completion.sh</markup>

<p>Once you have generated the script, you can source it.</p>

<markup
lang="bash"

>$ source ~/cohctl-completion.sh</markup>

</div>

<h3 id="_using_command_completion">Using Command Completion</h3>
<div class="section">
<p>Once your script is sourced, you can type <code>cohctl</code> and then press <code>TAB</code>
to auto-complete a command and <code>TAB</code> twice, and it will show completion
for a sub command.</p>

<p>Complete the <code>describe</code> command by typing the following and then <code>TAB</code>.</p>

<markup
lang="bash"

>$ cohctl des</markup>

<p>Next, when the following is displayed, press <code>TAB</code> twice, and you will see the available describe options.</p>

<markup
lang="bash"

>$ cohctl describe

cohctl describe
cache        (describe a cache)         machine      (describe a machine)       reporter     (describe a reporter)
cluster      (describe a cluster)       member       (describe a member)        service      (describe a service)
http-server  (describe a http server)   proxy        (describe a proxy server)</markup>

</div>

<h3 id="_command_completion_for_describe_commands">Command Completion for Describe Commands</h3>
<div class="section">
<p>You can use command completion on any <code>describe</code> operation and the CLI will display a list
of possible options. For example:</p>

<p>If you typed <code>cohctl describe cache</code>, then pressed <code>TAB</code> twice, it will show the list of caches that you can describe:</p>

<markup
lang="bash"

>$ cohctl describe cache
customers  orders     products</markup>

</div>
</div>
</doc-view>
