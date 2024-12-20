<doc-view>

<h2 id="_sorting_table_output">Sorting Table Output</h2>
<div class="section">
<p>By default, the table output is sorted by a meaningful column based upon the context. For example,
if we use the <code>cohctl get members</code> command, the output is sorted by the <code>NODE ID</code> column as shown below.</p>

<markup
lang="bash"

>cohctl get members -c main

Total cluster members: 3
Storage enabled count: 3
Departure count:       0

Cluster Heap - Total: 768 MB Used: 192 MB Available: 576 MB (75.0%)
Storage Heap - Total: 768 MB Used: 192 MB Available: 576 MB (75.0%)

NODE ID  ADDRESS     PORT   PROCESS  MEMBER     ROLE             STORAGE  MAX HEAP  USED HEAP  AVAIL HEAP
      1  /127.0.0.1  56125    34937  storage-2  CoherenceServer  true       256 MB      43 MB      213 MB
      2  /127.0.0.1  56124    34936  storage-1  CoherenceServer  true       256 MB      46 MB      210 MB
      3  /127.0.0.1  56126    34935  storage-0  CoherenceServer  true       256 MB     103 MB      153 MB</markup>


<h4 id="_specifying_a_custom_sort_column">Specifying a custom sort column</h4>
<div class="section">
<p>The CLI allows you to specify a custom sorting column for a table output by specifying the <code>--sort</code> option and
providing a column header name such as <code>ROLE</code> or <code>'AVAIL HEAP'</code>, or a column number starting from 1.
The sort is ascending by default, but can be chanegd to descending by specifying the <code>--desc</code> flag.</p>

<p>If the column is numerical, then it will be sorted as a number otherwise it will be sorted as a string.</p>

<div class="admonition note">
<p class="admonition-inline">If a column name has a space in it, you must surround the column name with single quotes.</p>
</div>
<p><strong>Example 1: Sort the members by Available Heap</strong></p>

<markup
lang="bash"

>cohctl get members -c local --sort 'AVAIL HEAP'

Total cluster members: 3
Storage enabled count: 3
Departure count:       0

Cluster Heap - Total: 768 MB Used: 209 MB Available: 559 MB (72.8%)
Storage Heap - Total: 768 MB Used: 209 MB Available: 559 MB (72.8%)

NODE ID  ADDRESS     PORT   PROCESS  MEMBER     ROLE             STORAGE  MAX HEAP  USED HEAP  AVAIL HEAP
      3  /127.0.0.1  56126    34935  storage-0  CoherenceServer  true       256 MB     112 MB      144 MB
      1  /127.0.0.1  56125    34937  storage-2  CoherenceServer  true       256 MB      51 MB      205 MB
      2  /127.0.0.1  56124    34936  storage-1  CoherenceServer  true       256 MB      46 MB      210 MB</markup>

<p><strong>Example 2: Sort the members by Available Heap descending</strong></p>

<markup
lang="bash"

>cohctl get members -c local --sort 'AVAIL HEAP' --desc

Total cluster members: 3
Storage enabled count: 3
Departure count:       0

Cluster Heap - Total: 768 MB Used: 225 MB Available: 543 MB (70.7%)
Storage Heap - Total: 768 MB Used: 225 MB Available: 543 MB (70.7%)

NODE ID  ADDRESS     PORT   PROCESS  MEMBER     ROLE             STORAGE  MAX HEAP  USED HEAP  AVAIL HEAP
      1  /127.0.0.1  56125    34937  storage-2  CoherenceServer  true       256 MB      53 MB      203 MB
      2  /127.0.0.1  56124    34936  storage-1  CoherenceServer  true       256 MB      55 MB      201 MB
      3  /127.0.0.1  56126    34935  storage-0  CoherenceServer  true       256 MB     117 MB      139 MB</markup>

<p><strong>Example 3: Sort the list of caches by COUNT descending</strong></p>

<div class="admonition note">
<p class="admonition-inline">In this example we specify the column number <code>3</code>, but you could also specify <code>COUNT</code>.</p>
</div>
<markup
lang="bash"

>cohctl get caches --sort 3 --desc
Using cluster connection 'main' from current context.

Total Caches: 3, Total primary storage: 33 MB

SERVICE           CACHE   COUNT   SIZE
PartitionedCache  test2  30,300  33 MB
PartitionedCache  test      100   0 MB
PartitionedCache  test3      10   0 MB</markup>

</div>
</div>
</doc-view>
