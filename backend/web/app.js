fetch('/api/findings').then(r=>r.json()).then(xs=>document.body.insertAdjacentHTML('beforeend','<ul>'+xs.map(x=>'<li>'+x.zone+': '+x.status+'</li>').join('')+'</ul>'));
