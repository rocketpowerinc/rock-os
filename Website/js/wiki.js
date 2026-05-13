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


const sidebarListContainer = document.getElementById('sidebarListContainer');
if (sidebarListContainer) {
	files.forEach(file => {
		const link = document.createElement('a');
		link.className = 'doc-link';
		link.href = '#';
		const cleanName = file.split('/').pop().replace('.md','');
		link.innerText = cleanName;
		link.onclick = () => { loadDoc(file); return false; };
		sidebarListContainer.appendChild(link);
	});
}

}


loadIndex();
