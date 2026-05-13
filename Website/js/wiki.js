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
	sidebarListContainer.innerHTML = `
		<h3>ARCHIVE</h3>
		<button class="cyber-button" onclick="refreshIndex()">REFRESH</button>
	`;
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

async function refreshIndex(){
await loadIndex();
}


loadIndex();

// Collapse/expand sidebar list
document.addEventListener('DOMContentLoaded', function() {
	const collapseBtn = document.getElementById('collapseListBtn');
	const sidebarList = document.getElementById('sidebarListContainer');
	let collapsed = false;
	if (collapseBtn && sidebarList) {
		collapseBtn.addEventListener('click', function() {
			collapsed = !collapsed;
			if (collapsed) {
				sidebarList.style.display = 'none';
				collapseBtn.innerHTML = '&#x25B6;'; // right arrow
				collapseBtn.setAttribute('aria-label', 'Expand List');
			} else {
				sidebarList.style.display = '';
				collapseBtn.innerHTML = '&#x25C0;'; // left arrow
				collapseBtn.setAttribute('aria-label', 'Collapse List');
			}
		});
	}
});

setInterval(loadIndex, 3000);
