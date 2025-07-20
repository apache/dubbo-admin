import{y as R,_ as D}from"./js-yaml-8Gkz3BRW.js";import{d as T,x as $,y as O,u as Y,z as h,W as L,F as w,G as v,w as e,e as s,o as b,b as t,f as a,n as E,a9 as M,aa as P,j as o,Q as j,m as I,p as z,h as J,_ as K}from"./index-hmLAZQYT.js";import{a as F}from"./traffic-C2a-KjHH.js";import"./request-8jI_GZey.js";const c=p=>(z("data-v-f0b44ffc"),p=p(),J(),p),G={class:"editorBox"},Q={class:"bottom-action-footer"},U=c(()=>o("br",null,null,-1)),W=c(()=>o("br",null,null,-1)),q=c(()=>o("br",null,null,-1)),H=c(()=>o("br",null,null,-1)),X=c(()=>o("br",null,null,-1)),Z=c(()=>o("br",null,null,-1)),ee=T({__name:"addByYAMLView",setup(p){const d=$(O.PROVIDE_INJECT_KEY),N=Y(),B=h(!1),r=h(!1),V=h(8),i=h(`conditions:
  - from:
      match: >-
        method=string & arguments[method]=string &
        arguments[arguments[method]]=string &
        arguments[arguments[arguments[method]]]=string &
        arguments[arguments[arguments[arguments[string]]]]!=string
    to:
      - match: string!=string
        weight: 0
  - from:
      match: >-
        method=string & arguments[method]=string &
        arguments[arguments[method]]=string &
        arguments[arguments[arguments[string]]]!=string
    to:
      - match: string!=lggbond
        weight: 0
      - match: ss!=ss
        weight: 0
configVersion: v3.1
enabled: true
force: false
key: org.apache.dubbo.samples.CommentService
runtime: true
scope: service`);L(()=>{if(w.isNil(d.conditionRule))i.value="";else{const l=d.conditionRule;i.value=R.dump(l)}});const S=l=>{d.conditionRule=R.load(i.value)},A=async()=>{const l=R.load(i.value),{configVersion:_,scope:g,key:u,runtime:x,force:C,conditions:y}=l;let n="";if(u=="application")n=`${u}.condition-router`;else if(w.isNil(d.addConditionRuleSate)){I.error("请先填写版本和分组字段");return}else{const{version:m,group:f}=d.addConditionRuleSate;if(m==""||f==""){I.error("请先填写版本和分组字段");return}n=`${u}:${m}:${f}.condition-router`}l.configVersion="v3.0",(await F(n,l)).code===200&&N.push("/traffic/routingRule")};return(l,_)=>{const g=s("a-button"),u=s("a-flex"),x=s("a-space"),C=s("a-affix"),y=s("a-col"),n=s("a-descriptions-item"),k=s("a-descriptions"),m=s("a-card");return b(),v(m,null,{default:e(()=>[t(u,{style:{width:"100%"}},{default:e(()=>[t(y,{span:r.value?24-V.value:24,class:"left"},{default:e(()=>[t(u,{vertical:"",align:"end"},{default:e(()=>[t(g,{type:"text",style:{color:"#0a90d5"},onClick:_[0]||(_[0]=f=>r.value=!r.value)},{default:e(()=>[a(" 字段说明 "),r.value?(b(),v(E(P),{key:1})):(b(),v(E(M),{key:0}))]),_:1}),o("div",G,[t(D,{onChange:S,modelValue:i.value,"onUpdate:modelValue":_[1]||(_[1]=f=>i.value=f),theme:"vs-dark",height:500,language:"yaml",readonly:B.value},null,8,["modelValue","readonly"])])]),_:1}),t(C,{"offset-bottom":10},{default:e(()=>[o("div",Q,[t(x,{align:"center",size:"large"},{default:e(()=>[t(g,{type:"primary",onClick:A},{default:e(()=>[a(" 确认 ")]),_:1}),t(g,null,{default:e(()=>[a(" 取消 ")]),_:1})]),_:1})])]),_:1})]),_:1},8,["span"]),t(y,{span:r.value?V.value:0,class:"right"},{default:e(()=>[r.value?(b(),v(m,{key:0,class:"sliderBox"},{default:e(()=>[o("div",null,[t(k,{title:"字段说明",column:1},{default:e(()=>[t(n,{label:"key"},{default:e(()=>[a(" 作用对象"),U,a(" 可能的值：Dubbo应用名或者服务名 ")]),_:1}),t(n,{label:"scope"},{default:e(()=>[a(" 规则粒度"),W,a(" 可能的值：application, service ")]),_:1}),t(n,{label:"force"},{default:e(()=>[a(" 容错保护"),q,a(" 可能的值：true, false"),H,a(" 描述：如果为true，则路由筛选后若没有可用的地址则会直接报异常；如果为false，则会从可用地址中选择完成RPC调用 ")]),_:1}),t(n,{label:"runtime"},{default:e(()=>[a(" 运行时生效"),X,a(" 可能的值：true, false"),Z,a(" 描述：如果为true，则该rule下的所有路由将会实时生效；若为false，则只有在启动时才会生效 ")]),_:1})]),_:1})])]),_:1})):j("",!0)]),_:1},8,["span"])]),_:1})]),_:1})}}}),se=K(ee,[["__scopeId","data-v-f0b44ffc"]]);export{se as default};
