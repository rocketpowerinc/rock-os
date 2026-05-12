async function loadDoc(path){

const response = await fetch(
path + '?nocache=' + Date.now()
);

if(!response.ok){
console.error('Failed:', path);
return;
}

const text = await response.text();

const md = window.markdownit({
html:true,
linkify:true,
typographer:true
});

document.getElementById('content').innerHTML =
md.render(text);

}

async function loadIndex(){

const response = await fetch(
'markdown-index.json?nocache=' + Date.now()
);

const files = await response.json();

const sidebar = document.getElementById('sidebar');

sidebar.innerHTML = `
<h3>ARCHIVE</h3>

<button class="cyber-button"
onclick="refreshIndex()">
REFRESH
</button>
`;

files.forEach(file => {

const link = document.createElement('a');

link.className = 'doc-link';

link.href = '#';

const cleanName = file
.split('/')
.pop()
.replace('.md','');

link.innerText = cleanName;

link.onclick = () => {
loadDoc(file);
return false;
};

sidebar.appendChild(link);

});

}

async function refreshIndex(){
await loadIndex();
}

loadIndex();

setInterval(loadIndex, 3000);
