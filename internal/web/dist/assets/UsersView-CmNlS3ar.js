import{u as oe}from"./useCrud-CFxH9XQZ.js";import{d as R,Z as Y,v as S,x as h,f as l,_ as ee,m as M,e as p,g as c,n as j,c as y,t as C,h as x,s as B,r as D,o as v,B as te,z as le,A as ae,$ as U,a0 as W,H as de,a1 as _,J as q,K as g,M as ue,N as E,Q as N,O,F as L,l as Q,a2 as ne,a3 as ce,b as pe,k as H,p as A,j as be}from"./index-FEfXL0mF.js";import{s as ve,c as fe,a as me,b as K}from"./index-BBIzS0bn.js";import{s as he}from"./index-BU1Ty1GK.js";import{s as J}from"./index-CfczcuHe.js";import{s as ge}from"./index-D89Kh0pJ.js";import{_ as re}from"./_plugin-vue_export-helper-DlAUqK2U.js";import{s as Z}from"./index-B9DgCcGe.js";import{s as ye}from"./index-Dgu92PY-.js";import"./index-yXPEAbXN.js";const we={class:"form-grid"},Te={class:"field"},ke={key:0,class:"p-error"},Pe={class:"field"},Ae={key:0,class:"p-error"},Ie={class:"field"},xe={for:"password"},Ce={key:0,class:"p-error"},Be={class:"field"},$e={class:"flex align-items-center gap-2"},Se={class:"dialog-footer"},De=R({__name:"UserForm",props:{visible:{type:Boolean},user:{}},emits:["close","save"],setup(t,{emit:e}){const a=t,r=e,i=D({name:"",email:"",password:"",active:!0}),n=D({}),f=M(()=>!!a.user),s=M(()=>f.value?"Edit User":"New User");Y(()=>a.visible,w=>{w&&(a.user?i.value={name:a.user.name,email:a.user.email,password:"",active:a.user.active}:i.value={name:"",email:"",password:"",active:!0},n.value={})});function u(){return n.value={},i.value.name.trim()||(n.value.name="Name is required"),i.value.email.trim()?/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(i.value.email)||(n.value.email="Invalid email format"):n.value.email="Email is required",!f.value&&!i.value.password?n.value.password="Password is required":i.value.password&&i.value.password.length<6&&(n.value.password="Password must be at least 6 characters"),Object.keys(n.value).length===0}function $(){if(u()){const w={name:i.value.name,email:i.value.email,password:i.value.password,active:i.value.active};f.value&&!i.value.password&&delete w.password,r("save",w)}}return(w,d)=>(v(),S(l(ee),{visible:t.visible,header:s.value,modal:"",closable:!0,style:{width:"450px"},"onUpdate:visible":d[5]||(d[5]=o=>w.$emit("close"))},{footer:h(()=>[p("div",Se,[c(l(B),{label:"Cancel",text:"",onClick:d[4]||(d[4]=o=>w.$emit("close"))}),c(l(B),{label:"Save",onClick:$})])]),default:h(()=>[p("div",we,[p("div",Te,[d[6]||(d[6]=p("label",{for:"name"},"Name",-1)),c(l(J),{id:"name",modelValue:i.value.name,"onUpdate:modelValue":d[0]||(d[0]=o=>i.value.name=o),class:j({"p-invalid":n.value.name}),placeholder:"Full name"},null,8,["modelValue","class"]),n.value.name?(v(),y("small",ke,C(n.value.name),1)):x("",!0)]),p("div",Pe,[d[7]||(d[7]=p("label",{for:"email"},"Email",-1)),c(l(J),{id:"email",modelValue:i.value.email,"onUpdate:modelValue":d[1]||(d[1]=o=>i.value.email=o),type:"email",class:j({"p-invalid":n.value.email}),placeholder:"Email address"},null,8,["modelValue","class"]),n.value.email?(v(),y("small",Ae,C(n.value.email),1)):x("",!0)]),p("div",Ie,[p("label",xe,"Password "+C(f.value?"(leave blank to keep)":""),1),c(l(ge),{id:"password",modelValue:i.value.password,"onUpdate:modelValue":d[2]||(d[2]=o=>i.value.password=o),class:j({"p-invalid":n.value.password}),feedback:!1,"toggle-mask":"",placeholder:"Password","input-style":{width:"100%"}},null,8,["modelValue","class"]),n.value.password?(v(),y("small",Ce,C(n.value.password),1)):x("",!0)]),p("div",Be,[p("div",$e,[c(l(ve),{modelValue:i.value.active,"onUpdate:modelValue":d[3]||(d[3]=o=>i.value.active=o),"input-id":"active",binary:!0},null,8,["modelValue"]),d[8]||(d[8]=p("label",{for:"active"},"Active",-1))])])])]),_:1},8,["visible","header"]))}}),Ve=re(De,[["__scopeId","data-v-ba172dbc"]]);var Ke=`
    .p-tabview-tablist-container {
        position: relative;
    }

    .p-tabview-scrollable > .p-tabview-tablist-container {
        overflow: hidden;
    }

    .p-tabview-tablist-scroll-container {
        overflow-x: auto;
        overflow-y: hidden;
        scroll-behavior: smooth;
        scrollbar-width: none;
        overscroll-behavior: contain auto;
    }

    .p-tabview-tablist-scroll-container::-webkit-scrollbar {
        display: none;
    }

    .p-tabview-tablist {
        display: flex;
        margin: 0;
        padding: 0;
        list-style-type: none;
        flex: 1 1 auto;
        background: dt('tabview.tab.list.background');
        border: 1px solid dt('tabview.tab.list.border.color');
        border-width: 0 0 1px 0;
        position: relative;
    }

    .p-tabview-tab-header {
        cursor: pointer;
        user-select: none;
        display: flex;
        align-items: center;
        text-decoration: none;
        position: relative;
        overflow: hidden;
        border-style: solid;
        border-width: 0 0 1px 0;
        border-color: transparent transparent dt('tabview.tab.border.color') transparent;
        color: dt('tabview.tab.color');
        padding: 1rem 1.125rem;
        font-weight: 600;
        border-top-right-radius: dt('border.radius.md');
        border-top-left-radius: dt('border.radius.md');
        transition:
            color dt('tabview.transition.duration'),
            outline-color dt('tabview.transition.duration');
        margin: 0 0 -1px 0;
        outline-color: transparent;
    }

    .p-tabview-tablist-item:not(.p-disabled) .p-tabview-tab-header:focus-visible {
        outline: dt('focus.ring.width') dt('focus.ring.style') dt('focus.ring.color');
        outline-offset: -1px;
    }

    .p-tabview-tablist-item:not(.p-highlight):not(.p-disabled):hover > .p-tabview-tab-header {
        color: dt('tabview.tab.hover.color');
    }

    .p-tabview-tablist-item.p-highlight > .p-tabview-tab-header {
        color: dt('tabview.tab.active.color');
    }

    .p-tabview-tab-title {
        line-height: 1;
        white-space: nowrap;
    }

    .p-tabview-next-button,
    .p-tabview-prev-button {
        position: absolute;
        top: 0;
        margin: 0;
        padding: 0;
        z-index: 2;
        height: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
        background: dt('tabview.nav.button.background');
        color: dt('tabview.nav.button.color');
        width: 2.5rem;
        border-radius: 0;
        outline-color: transparent;
        transition:
            color dt('tabview.transition.duration'),
            outline-color dt('tabview.transition.duration');
        box-shadow: dt('tabview.nav.button.shadow');
        border: none;
        cursor: pointer;
        user-select: none;
    }

    .p-tabview-next-button:focus-visible,
    .p-tabview-prev-button:focus-visible {
        outline: dt('focus.ring.width') dt('focus.ring.style') dt('focus.ring.color');
        outline-offset: dt('focus.ring.offset');
    }

    .p-tabview-next-button:hover,
    .p-tabview-prev-button:hover {
        color: dt('tabview.nav.button.hover.color');
    }

    .p-tabview-prev-button {
        left: 0;
    }

    .p-tabview-next-button {
        right: 0;
    }

    .p-tabview-panels {
        background: dt('tabview.tab.panel.background');
        color: dt('tabview.tab.panel.color');
        padding: 0.875rem 1.125rem 1.125rem 1.125rem;
    }

    .p-tabview-ink-bar {
        z-index: 1;
        display: block;
        position: absolute;
        bottom: -1px;
        height: 1px;
        background: dt('tabview.tab.active.border.color');
        transition: 250ms cubic-bezier(0.35, 0, 0.25, 1);
    }
`,Ue={root:function(e){var a=e.props;return["p-tabview p-component",{"p-tabview-scrollable":a.scrollable}]},navContainer:"p-tabview-tablist-container",prevButton:"p-tabview-prev-button",navContent:"p-tabview-tablist-scroll-container",nav:"p-tabview-tablist",tab:{header:function(e){var a=e.instance,r=e.tab,i=e.index;return["p-tabview-tablist-item",a.getTabProp(r,"headerClass"),{"p-tabview-tablist-item-active":a.d_activeIndex===i,"p-disabled":a.getTabProp(r,"disabled")}]},headerAction:"p-tabview-tab-header",headerTitle:"p-tabview-tab-title",content:function(e){var a=e.instance,r=e.tab;return["p-tabview-panel",a.getTabProp(r,"contentClass")]}},inkbar:"p-tabview-ink-bar",nextButton:"p-tabview-next-button",panelContainer:"p-tabview-panels"},Ee=te.extend({name:"tabview",style:Ke,classes:Ue}),Oe={name:"BaseTabView",extends:ae,props:{activeIndex:{type:Number,default:0},lazy:{type:Boolean,default:!1},scrollable:{type:Boolean,default:!1},tabindex:{type:Number,default:0},selectOnFocus:{type:Boolean,default:!1},prevButtonProps:{type:null,default:null},nextButtonProps:{type:null,default:null},prevIcon:{type:String,default:void 0},nextIcon:{type:String,default:void 0}},style:Ee,provide:function(){return{$pcTabs:void 0,$pcTabView:this,$parentInstance:this}}},ie={name:"TabView",extends:Oe,inheritAttrs:!1,emits:["update:activeIndex","tab-change","tab-click"],data:function(){return{d_activeIndex:this.activeIndex,isPrevButtonDisabled:!0,isNextButtonDisabled:!1}},watch:{activeIndex:function(e){this.d_activeIndex=e,this.scrollInView({index:e})}},mounted:function(){console.warn("Deprecated since v4. Use Tabs component instead."),this.updateInkBar(),this.scrollable&&this.updateButtonState()},updated:function(){this.updateInkBar(),this.scrollable&&this.updateButtonState()},methods:{isTabPanel:function(e){return e.type.name==="TabPanel"},isTabActive:function(e){return this.d_activeIndex===e},getTabProp:function(e,a){return e.props?e.props[a]:void 0},getKey:function(e,a){return this.getTabProp(e,"header")||a},getTabHeaderActionId:function(e){return"".concat(this.$id,"_").concat(e,"_header_action")},getTabContentId:function(e){return"".concat(this.$id,"_").concat(e,"_content")},getTabPT:function(e,a,r){var i=this.tabs.length,n={props:e.props,parent:{instance:this,props:this.$props,state:this.$data},context:{index:r,count:i,first:r===0,last:r===i-1,active:this.isTabActive(r)}};return g(this.ptm("tabpanel.".concat(a),{tabpanel:n}),this.ptm("tabpanel.".concat(a),n),this.ptmo(this.getTabProp(e,"pt"),a,n))},onScroll:function(e){this.scrollable&&this.updateButtonState(),e.preventDefault()},onPrevButtonClick:function(){var e=this.$refs.content,a=U(e),r=e.scrollLeft-a;e.scrollLeft=r<=0?0:r},onNextButtonClick:function(){var e=this.$refs.content,a=U(e)-this.getVisibleButtonWidths(),r=e.scrollLeft+a,i=e.scrollWidth-a;e.scrollLeft=r>=i?i:r},onTabClick:function(e,a,r){this.changeActiveIndex(e,a,r),this.$emit("tab-click",{originalEvent:e,index:r})},onTabKeyDown:function(e,a,r){switch(e.code){case"ArrowLeft":this.onTabArrowLeftKey(e);break;case"ArrowRight":this.onTabArrowRightKey(e);break;case"Home":this.onTabHomeKey(e);break;case"End":this.onTabEndKey(e);break;case"PageDown":this.onPageDownKey(e);break;case"PageUp":this.onPageUpKey(e);break;case"Enter":case"NumpadEnter":case"Space":this.onTabEnterKey(e,a,r);break}},onTabArrowRightKey:function(e){var a=this.findNextHeaderAction(e.target.parentElement);a?this.changeFocusedTab(e,a):this.onTabHomeKey(e),e.preventDefault()},onTabArrowLeftKey:function(e){var a=this.findPrevHeaderAction(e.target.parentElement);a?this.changeFocusedTab(e,a):this.onTabEndKey(e),e.preventDefault()},onTabHomeKey:function(e){var a=this.findFirstHeaderAction();this.changeFocusedTab(e,a),e.preventDefault()},onTabEndKey:function(e){var a=this.findLastHeaderAction();this.changeFocusedTab(e,a),e.preventDefault()},onPageDownKey:function(e){this.scrollInView({index:this.$refs.nav.children.length-2}),e.preventDefault()},onPageUpKey:function(e){this.scrollInView({index:0}),e.preventDefault()},onTabEnterKey:function(e,a,r){this.changeActiveIndex(e,a,r),e.preventDefault()},findNextHeaderAction:function(e){var a=arguments.length>1&&arguments[1]!==void 0?arguments[1]:!1,r=a?e:e.nextElementSibling;return r?_(r,"data-p-disabled")||_(r,"data-pc-section")==="inkbar"?this.findNextHeaderAction(r):q(r,'[data-pc-section="headeraction"]'):null},findPrevHeaderAction:function(e){var a=arguments.length>1&&arguments[1]!==void 0?arguments[1]:!1,r=a?e:e.previousElementSibling;return r?_(r,"data-p-disabled")||_(r,"data-pc-section")==="inkbar"?this.findPrevHeaderAction(r):q(r,'[data-pc-section="headeraction"]'):null},findFirstHeaderAction:function(){return this.findNextHeaderAction(this.$refs.nav.firstElementChild,!0)},findLastHeaderAction:function(){return this.findPrevHeaderAction(this.$refs.nav.lastElementChild,!0)},changeActiveIndex:function(e,a,r){!this.getTabProp(a,"disabled")&&this.d_activeIndex!==r&&(this.d_activeIndex=r,this.$emit("update:activeIndex",r),this.$emit("tab-change",{originalEvent:e,index:r}),this.scrollInView({index:r}))},changeFocusedTab:function(e,a){if(a&&(de(a),this.scrollInView({element:a}),this.selectOnFocus)){var r=parseInt(a.parentElement.dataset.pcIndex,10),i=this.tabs[r];this.changeActiveIndex(e,i,r)}},scrollInView:function(e){var a=e.element,r=e.index,i=r===void 0?-1:r,n=a||this.$refs.nav.children[i];n&&n.scrollIntoView&&n.scrollIntoView({block:"nearest"})},updateInkBar:function(){var e=this.$refs.nav.children[this.d_activeIndex];this.$refs.inkbar.style.width=U(e)+"px",this.$refs.inkbar.style.left=W(e).left-W(this.$refs.nav).left+"px"},updateButtonState:function(){var e=this.$refs.content,a=e.scrollLeft,r=e.scrollWidth,i=U(e);this.isPrevButtonDisabled=a===0,this.isNextButtonDisabled=parseInt(a)===r-i},getVisibleButtonWidths:function(){var e=this.$refs,a=e.prevBtn,r=e.nextBtn;return[a,r].reduce(function(i,n){return n?i+U(n):i},0)}},computed:{tabs:function(){var e=this;return this.$slots.default().reduce(function(a,r){return e.isTabPanel(r)?a.push(r):r.children&&r.children instanceof Array&&r.children.forEach(function(i){e.isTabPanel(i)&&a.push(i)}),a},[])},prevButtonAriaLabel:function(){return this.$primevue.config.locale.aria?this.$primevue.config.locale.aria.previous:void 0},nextButtonAriaLabel:function(){return this.$primevue.config.locale.aria?this.$primevue.config.locale.aria.next:void 0}},directives:{ripple:le},components:{ChevronLeftIcon:ye,ChevronRightIcon:fe}};function z(t){"@babel/helpers - typeof";return z=typeof Symbol=="function"&&typeof Symbol.iterator=="symbol"?function(e){return typeof e}:function(e){return e&&typeof Symbol=="function"&&e.constructor===Symbol&&e!==Symbol.prototype?"symbol":typeof e},z(t)}function X(t,e){var a=Object.keys(t);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(t);e&&(r=r.filter(function(i){return Object.getOwnPropertyDescriptor(t,i).enumerable})),a.push.apply(a,r)}return a}function P(t){for(var e=1;e<arguments.length;e++){var a=arguments[e]!=null?arguments[e]:{};e%2?X(Object(a),!0).forEach(function(r){He(t,r,a[r])}):Object.getOwnPropertyDescriptors?Object.defineProperties(t,Object.getOwnPropertyDescriptors(a)):X(Object(a)).forEach(function(r){Object.defineProperty(t,r,Object.getOwnPropertyDescriptor(a,r))})}return t}function He(t,e,a){return(e=Ne(e))in t?Object.defineProperty(t,e,{value:a,enumerable:!0,configurable:!0,writable:!0}):t[e]=a,t}function Ne(t){var e=Le(t,"string");return z(e)=="symbol"?e:e+""}function Le(t,e){if(z(t)!="object"||!t)return t;var a=t[Symbol.toPrimitive];if(a!==void 0){var r=a.call(t,e);if(z(r)!="object")return r;throw new TypeError("@@toPrimitive must return a primitive value.")}return(e==="string"?String:Number)(t)}var ze=["tabindex","aria-label"],Fe=["data-p-active","data-p-disabled","data-pc-index"],_e=["id","tabindex","aria-disabled","aria-selected","aria-controls","onClick","onKeydown"],je=["tabindex","aria-label"],Me=["id","aria-labelledby","data-pc-index","data-p-active"];function Ge(t,e,a,r,i,n){var f=ue("ripple");return v(),y("div",g({class:t.cx("root"),role:"tablist"},t.ptmi("root")),[p("div",g({class:t.cx("navContainer")},t.ptm("navContainer")),[t.scrollable&&!i.isPrevButtonDisabled?E((v(),y("button",g({key:0,ref:"prevBtn",type:"button",class:t.cx("prevButton"),tabindex:t.tabindex,"aria-label":n.prevButtonAriaLabel,onClick:e[0]||(e[0]=function(){return n.onPrevButtonClick&&n.onPrevButtonClick.apply(n,arguments)})},P(P({},t.prevButtonProps),t.ptm("prevButton")),{"data-pc-group-section":"navbutton"}),[N(t.$slots,"previcon",{},function(){return[(v(),S(O(t.prevIcon?"span":"ChevronLeftIcon"),g({"aria-hidden":"true",class:t.prevIcon},t.ptm("prevIcon")),null,16,["class"]))]})],16,ze)),[[f]]):x("",!0),p("div",g({ref:"content",class:t.cx("navContent"),onScroll:e[1]||(e[1]=function(){return n.onScroll&&n.onScroll.apply(n,arguments)})},t.ptm("navContent")),[p("ul",g({ref:"nav",class:t.cx("nav")},t.ptm("nav")),[(v(!0),y(L,null,Q(n.tabs,function(s,u){return v(),y("li",g({key:n.getKey(s,u),style:n.getTabProp(s,"headerStyle"),class:t.cx("tab.header",{tab:s,index:u}),role:"presentation"},{ref_for:!0},P(P(P({},n.getTabProp(s,"headerProps")),n.getTabPT(s,"root",u)),n.getTabPT(s,"header",u)),{"data-pc-name":"tabpanel","data-p-active":i.d_activeIndex===u,"data-p-disabled":n.getTabProp(s,"disabled"),"data-pc-index":u}),[E((v(),y("a",g({id:n.getTabHeaderActionId(u),class:t.cx("tab.headerAction"),tabindex:n.getTabProp(s,"disabled")||!n.isTabActive(u)?-1:t.tabindex,role:"tab","aria-disabled":n.getTabProp(s,"disabled"),"aria-selected":n.isTabActive(u),"aria-controls":n.getTabContentId(u),onClick:function(w){return n.onTabClick(w,s,u)},onKeydown:function(w){return n.onTabKeyDown(w,s,u)}},{ref_for:!0},P(P({},n.getTabProp(s,"headerActionProps")),n.getTabPT(s,"headerAction",u))),[s.props&&s.props.header?(v(),y("span",g({key:0,class:t.cx("tab.headerTitle")},{ref_for:!0},n.getTabPT(s,"headerTitle",u)),C(s.props.header),17)):x("",!0),s.children&&s.children.header?(v(),S(O(s.children.header),{key:1})):x("",!0)],16,_e)),[[f]])],16,Fe)}),128)),p("li",g({ref:"inkbar",class:t.cx("inkbar"),role:"presentation","aria-hidden":"true"},t.ptm("inkbar")),null,16)],16)],16),t.scrollable&&!i.isNextButtonDisabled?E((v(),y("button",g({key:1,ref:"nextBtn",type:"button",class:t.cx("nextButton"),tabindex:t.tabindex,"aria-label":n.nextButtonAriaLabel,onClick:e[2]||(e[2]=function(){return n.onNextButtonClick&&n.onNextButtonClick.apply(n,arguments)})},P(P({},t.nextButtonProps),t.ptm("nextButton")),{"data-pc-group-section":"navbutton"}),[N(t.$slots,"nexticon",{},function(){return[(v(),S(O(t.nextIcon?"span":"ChevronRightIcon"),g({"aria-hidden":"true",class:t.nextIcon},t.ptm("nextIcon")),null,16,["class"]))]})],16,je)),[[f]]):x("",!0)],16),p("div",g({class:t.cx("panelContainer")},t.ptm("panelContainer")),[(v(!0),y(L,null,Q(n.tabs,function(s,u){return v(),y(L,{key:n.getKey(s,u)},[!t.lazy||n.isTabActive(u)?E((v(),y("div",g({key:0,id:n.getTabContentId(u),style:n.getTabProp(s,"contentStyle"),class:t.cx("tab.content",{tab:s}),role:"tabpanel","aria-labelledby":n.getTabHeaderActionId(u)},{ref_for:!0},P(P(P({},n.getTabProp(s,"contentProps")),n.getTabPT(s,"root",u)),n.getTabPT(s,"content",u)),{"data-pc-name":"tabpanel","data-pc-index":u,"data-p-active":i.d_activeIndex===u}),[(v(),S(O(s)))],16,Me)),[[ne,t.lazy?!0:n.isTabActive(u)]]):x("",!0)],64)}),128))],16)],16)}ie.render=Ge;var Re={root:function(e){var a=e.instance;return["p-tabpanel",{"p-tabpanel-active":a.active}]}},We=te.extend({name:"tabpanel",classes:Re}),qe={name:"BaseTabPanel",extends:ae,props:{value:{type:[String,Number],default:void 0},as:{type:[String,Object],default:"DIV"},asChild:{type:Boolean,default:!1},header:null,headerStyle:null,headerClass:null,headerProps:null,headerActionProps:null,contentStyle:null,contentClass:null,contentProps:null,disabled:Boolean},style:We,provide:function(){return{$pcTabPanel:this,$parentInstance:this}}},G={name:"TabPanel",extends:qe,inheritAttrs:!1,inject:["$pcTabs"],computed:{active:function(){var e;return ce((e=this.$pcTabs)===null||e===void 0?void 0:e.d_value,this.value)},id:function(){var e;return"".concat((e=this.$pcTabs)===null||e===void 0?void 0:e.$id,"_tabpanel_").concat(this.value)},ariaLabelledby:function(){var e;return"".concat((e=this.$pcTabs)===null||e===void 0?void 0:e.$id,"_tab_").concat(this.value)},attrs:function(){return g(this.a11yAttrs,this.ptmi("root",this.ptParams))},a11yAttrs:function(){var e;return{id:this.id,tabindex:(e=this.$pcTabs)===null||e===void 0?void 0:e.tabindex,role:"tabpanel","aria-labelledby":this.ariaLabelledby,"data-pc-name":"tabpanel","data-p-active":this.active}},ptParams:function(){return{context:{active:this.active}}}}};function Qe(t,e,a,r,i,n){var f,s;return n.$pcTabs?(v(),y(L,{key:1},[t.asChild?N(t.$slots,"default",{key:1,class:j(t.cx("root")),active:n.active,a11yAttrs:n.a11yAttrs}):(v(),y(L,{key:0},[!((f=n.$pcTabs)!==null&&f!==void 0&&f.lazy)||n.active?E((v(),S(O(t.as),g({key:0,class:t.cx("root")},n.attrs),{default:h(function(){return[N(t.$slots,"default")]}),_:3},16,["class"])),[[ne,(s=n.$pcTabs)!==null&&s!==void 0&&s.lazy?!0:n.active]]):x("",!0)],64))],64)):N(t.$slots,"default",{key:0})}G.render=Qe;const Je={class:"dialog-footer"},Ze=R({__name:"UserMembershipsDialog",props:{visible:{type:Boolean},user:{}},emits:["close","updated"],setup(t,{emit:e}){const a=t,r=e,i=pe(),n=D(!1),f=D([[],[]]),s=D([[],[]]),u=M(()=>a.user?`Manage Memberships: ${a.user.name}`:"Manage Memberships");Y(()=>a.visible,async d=>{d&&a.user&&await $()});async function $(){if(a.user){n.value=!0;try{const[d,o,m]=await Promise.all([A.getGroups(),A.getOrganizations(),A.getUser(a.user.id)]),V=new Set((m.groups||[]).map(T=>T.id)),k=new Set((m.organizations||[]).map(T=>T.id));f.value=[d.filter(T=>!V.has(T.id)),m.groups||[]],s.value=[o.filter(T=>!k.has(T.id)),m.organizations||[]]}catch{i.add({severity:"error",summary:"Error",detail:"Failed to load membership data",life:3e3})}finally{n.value=!1}}}async function w(){if(a.user){n.value=!0;try{const d=await A.getUser(a.user.id),o=new Set((d.groups||[]).map(b=>b.id)),m=new Set((d.organizations||[]).map(b=>b.id)),V=new Set(f.value[1].map(b=>b.id)),k=new Set(s.value[1].map(b=>b.id)),T=[...V].filter(b=>!o.has(b)),I=[...o].filter(b=>!V.has(b)),F=[...k].filter(b=>!m.has(b)),se=[...m].filter(b=>!k.has(b));await Promise.all([...T.map(b=>A.addUserToGroup(a.user.id,b)),...I.map(b=>A.removeUserFromGroup(a.user.id,b)),...F.map(b=>A.addUserToOrganization(a.user.id,b)),...se.map(b=>A.removeUserFromOrganization(a.user.id,b))]),i.add({severity:"success",summary:"Success",detail:"Memberships updated successfully",life:3e3}),r("updated"),r("close")}catch{i.add({severity:"error",summary:"Error",detail:"Failed to update memberships",life:3e3})}finally{n.value=!1}}}return(d,o)=>(v(),S(l(ee),{visible:t.visible,header:u.value,modal:"",closable:!0,style:{width:"700px"},"onUpdate:visible":o[3]||(o[3]=m=>d.$emit("close"))},{footer:h(()=>[p("div",Je,[c(l(B),{label:"Cancel",text:"",onClick:o[2]||(o[2]=m=>d.$emit("close"))}),c(l(B),{label:"Save",loading:n.value,onClick:w},null,8,["loading"])])]),default:h(()=>[c(l(ie),null,{default:h(()=>[c(l(G),{value:"groups",header:"Groups"},{default:h(()=>[o[6]||(o[6]=p("p",{class:"mb-3"},"Move groups between Available and Assigned lists:",-1)),c(l(Z),{modelValue:f.value,"onUpdate:modelValue":o[0]||(o[0]=m=>f.value=m),"data-key":"id",breakpoint:"575px","show-source-controls":!1,"show-target-controls":!1},{sourceheader:h(()=>[...o[4]||(o[4]=[H("Available Groups",-1)])]),targetheader:h(()=>[...o[5]||(o[5]=[H("Assigned Groups",-1)])]),item:h(({item:m})=>[p("span",null,C(m.name),1)]),_:1},8,["modelValue"])]),_:1}),c(l(G),{value:"organizations",header:"Organizations"},{default:h(()=>[o[9]||(o[9]=p("p",{class:"mb-3"},"Move organizations between Available and Assigned lists:",-1)),c(l(Z),{modelValue:s.value,"onUpdate:modelValue":o[1]||(o[1]=m=>s.value=m),"data-key":"id",breakpoint:"575px","show-source-controls":!1,"show-target-controls":!1},{sourceheader:h(()=>[...o[7]||(o[7]=[H("Available Organizations",-1)])]),targetheader:h(()=>[...o[8]||(o[8]=[H("Assigned Organizations",-1)])]),item:h(({item:m})=>[p("span",null,C(m.name),1)]),_:1},8,["modelValue"])]),_:1})]),_:1})]),_:1},8,["visible","header"]))}}),Xe=re(Ze,[["__scopeId","data-v-d97badba"]]),Ye={class:"page-header"},et={class:"card"},ct=R({__name:"UsersView",setup(t){const{items:e,loading:a,dialogVisible:r,editingItem:i,fetchItems:n,openCreateDialog:f,openEditDialog:s,closeDialog:u,saveItem:$,confirmDelete:w}=oe({entityName:"User",fetchAll:()=>A.getUsers(),create:k=>A.createUser(k),update:(k,T)=>A.updateUser(k,T),remove:k=>A.deleteUser(k)}),d=D(!1),o=D(null);function m(k){o.value=k,d.value=!0}function V(){d.value=!1,o.value=null}return be(()=>{n()}),(k,T)=>(v(),y("div",null,[p("div",Ye,[T[0]||(T[0]=p("h1",null,"Users",-1)),c(l(B),{label:"New User",icon:"pi pi-plus",onClick:l(f)},null,8,["onClick"])]),p("div",et,[c(l(me),{value:l(e),loading:l(a),"striped-rows":"",paginator:"",rows:10,"rows-per-page-options":[10,25,50]},{default:h(()=>[c(l(K),{field:"id",header:"ID",sortable:"",style:{width:"80px"}}),c(l(K),{field:"name",header:"Name",sortable:""}),c(l(K),{field:"email",header:"Email",sortable:""}),c(l(K),{field:"active",header:"Status",sortable:"",style:{width:"120px"}},{body:h(({data:I})=>[c(l(he),{value:I.active?"Active":"Inactive",severity:I.active?"success":"danger"},null,8,["value","severity"])]),_:1}),c(l(K),{field:"created_at",header:"Created",sortable:"",style:{width:"180px"}},{body:h(({data:I})=>[H(C(new Date(I.created_at).toLocaleDateString()),1)]),_:1}),c(l(K),{header:"Actions",style:{width:"200px"}},{body:h(({data:I})=>[c(l(B),{icon:"pi pi-users",text:"",rounded:"",title:"Manage Memberships",onClick:F=>m(I)},null,8,["onClick"]),c(l(B),{icon:"pi pi-pencil",text:"",rounded:"",title:"Edit",onClick:F=>l(s)(I)},null,8,["onClick"]),c(l(B),{icon:"pi pi-trash",text:"",rounded:"",severity:"danger",title:"Delete",onClick:F=>l(w)(I)},null,8,["onClick"])]),_:1})]),_:1},8,["value","loading"])]),c(Ve,{visible:l(r),user:l(i),onClose:l(u),onSave:l($)},null,8,["visible","user","onClose","onSave"]),c(Xe,{visible:d.value,user:o.value,onClose:V,onUpdated:l(n)},null,8,["visible","user","onUpdated"])]))}});export{ct as default};
