import{y as R,_ as T}from"./js-yaml-EQlPfOK8.js";import{d as A,y as $,z as O,u as Y,B as h,D as L,H as w,J as v,w as e,e as s,o as b,b as t,f as a,n as B,aa as M,ab as P,j as o,T as J,m as E,p as j,h as z,_ as K}from"./index-VXjVsiiO.js";import{a as H}from"./traffic-W0fp5Gf-.js";import"./request-Cs8TyifY.js";const c=p=>(j("data-v-f0b44ffc"),p=p(),z(),p),U={class:"editorBox"},q={class:"bottom-action-footer"},F=c(()=>o("br",null,null,-1)),G=c(()=>o("br",null,null,-1)),Q=c(()=>o("br",null,null,-1)),W=c(()=>o("br",null,null,-1)),X=c(()=>o("br",null,null,-1)),Z=c(()=>o("br",null,null,-1)),ee=A({__name:"addByYAMLView",setup(p){const d=$(O.PROVIDE_INJECT_KEY),I=Y(),N=h(!1),r=h(!1),V=h(8),i=h(`conditions:
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
scope: service`);L(()=>{if(w.isNil(d.conditionRule))i.value="";else{const l=d.conditionRule;i.value=R.dump(l)}});const S=l=>{d.conditionRule=R.load(i.value)},D=async()=>{const l=R.load(i.value),{configVersion:_,scope:g,key:u,runtime:C,force:x,conditions:y}=l;let n="";if(u=="application")n=`${u}.condition-router`;else if(w.isNil(d.addConditionRuleSate)){E.error("请先填写版本和分组字段");return}else{const{version:m,group:f}=d.addConditionRuleSate;if(m==""||f==""){E.error("请先填写版本和分组字段");return}n=`${u}:${m}:${f}.condition-router`}l.configVersion="v3.0",(await H(n,l)).code===200&&I.push("/traffic/routingRule")};return(l,_)=>{const g=s("a-button"),u=s("a-flex"),C=s("a-space"),x=s("a-affix"),y=s("a-col"),n=s("a-descriptions-item"),k=s("a-descriptions"),m=s("a-card");return b(),v(m,null,{default:e(()=>[t(u,{style:{width:"100%"}},{default:e(()=>[t(y,{span:r.value?24-V.value:24,class:"left"},{default:e(()=>[t(u,{vertical:"",align:"end"},{default:e(()=>[t(g,{type:"text",style:{color:"#0a90d5"},onClick:_[0]||(_[0]=f=>r.value=!r.value)},{default:e(()=>[a(" 字段说明 "),r.value?(b(),v(B(P),{key:1})):(b(),v(B(M),{key:0}))]),_:1}),o("div",U,[t(T,{onChange:S,modelValue:i.value,"onUpdate:modelValue":_[1]||(_[1]=f=>i.value=f),theme:"vs-dark",height:500,language:"yaml",readonly:N.value},null,8,["modelValue","readonly"])])]),_:1}),t(x,{"offset-bottom":10},{default:e(()=>[o("div",q,[t(C,{align:"center",size:"large"},{default:e(()=>[t(g,{type:"primary",onClick:D},{default:e(()=>[a(" 确认 ")]),_:1}),t(g,null,{default:e(()=>[a(" 取消 ")]),_:1})]),_:1})])]),_:1})]),_:1},8,["span"]),t(y,{span:r.value?V.value:0,class:"right"},{default:e(()=>[r.value?(b(),v(m,{key:0,class:"sliderBox"},{default:e(()=>[o("div",null,[t(k,{title:"字段说明",column:1},{default:e(()=>[t(n,{label:"key"},{default:e(()=>[a(" 作用对象"),F,a(" 可能的值：Dubbo应用名或者服务名 ")]),_:1}),t(n,{label:"scope"},{default:e(()=>[a(" 规则粒度"),G,a(" 可能的值：application, service ")]),_:1}),t(n,{label:"force"},{default:e(()=>[a(" 容错保护"),Q,a(" 可能的值：true, false"),W,a(" 描述：如果为true，则路由筛选后若没有可用的地址则会直接报异常；如果为false，则会从可用地址中选择完成RPC调用 ")]),_:1}),t(n,{label:"runtime"},{default:e(()=>[a(" 运行时生效"),X,a(" 可能的值：true, false"),Z,a(" 描述：如果为true，则该rule下的所有路由将会实时生效；若为false，则只有在启动时才会生效 ")]),_:1})]),_:1})])]),_:1})):J("",!0)]),_:1},8,["span"])]),_:1})]),_:1})}}}),se=K(ee,[["__scopeId","data-v-f0b44ffc"]]);export{se as default};
