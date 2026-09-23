import { defineStore } from 'pinia';
import { dashboardItems } from '../data/sample.data';
import { saveItems } from '../services/storage.service';
export const useDashboardStore=defineStore('dashboard',{state:()=>({items:dashboardItems,keyword:''}),getters:{filtered:(state)=>state.items.filter(item=>(item.title+item.description+item.status).includes(state.keyword))},actions:{advance(id:string){this.items=this.items.map(item=>item.id===id?{...item,status:item.status==='已完成'?'待处理':'已完成',score:Math.min(100,item.score+5)}:item);saveItems(this.items)}}});