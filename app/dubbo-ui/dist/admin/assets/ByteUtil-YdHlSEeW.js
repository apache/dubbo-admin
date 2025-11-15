function u(n=0){const o=["B","KB","MB","GB","TB"];let t=0;for(;n>=1024&&t<o.length-1;)n/=1024,t++;return`${Math.round(n)} ${o[t]}`}export{u as b};
