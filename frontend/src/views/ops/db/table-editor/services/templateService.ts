/**
 * 模板系统 - TemplateService
 * 内置模板数据已拆分到 templateData.ts
 */

import type { TableTemplate, TableDefinition } from '../../types/schema';
import { i18n } from '@/i18n';
import { builtinTemplateData } from './templateData';

class TemplateService {
  /**
   * 本地化内置模板：模板数据的可翻译字段（name/description/tags/列注释/索引注释）
   * 均直接存放 i18n key，此处统一 t() 翻译为当前语言文本。
   *
   * 仅用于内置模板；用户自定义模板保存的即是明文，不经此处理。
   */
  private localizeTemplate(tpl: TableTemplate): TableTemplate {
    const t = i18n.global.t;
    return {
      ...tpl,
      name: t(tpl.name),
      description: t(tpl.description),
      tags: tpl.tags.map((k) => t(k)),
      columns: tpl.columns.map((col) => ({ ...col, comment: col.comment ? t(col.comment) : col.comment })),
      indexes: tpl.indexes?.map((idx) => ({ ...idx, comment: idx.comment ? t(idx.comment) : idx.comment })),
    };
  }

  /**
   * 获取所有模板
   */
  getAllTemplates(): TableTemplate[] {
    return [
      ...builtinTemplateData.map(tpl => this.localizeTemplate(tpl)),
      ...this.getUserTemplates(),
    ];
  }
  
  /**
   * 获取内置模板
   */
  getBuiltinTemplates(): TableTemplate[] {
    return builtinTemplateData.map(tpl => this.localizeTemplate(tpl));
  }
  
  /**
   * 获取用户自定义模板
   */
  getUserTemplates(): TableTemplate[] {
    try {
      const stored = localStorage.getItem('user_table_templates');
      return stored ? JSON.parse(stored) : [];
    } catch (error) {
      console.error('获取用户模板失败:', error);
      return [];
    }
  }
  
  /**
   * 根据 ID 获取模板
   */
  getTemplateById(id: string): TableTemplate | null {
    const all = this.getAllTemplates();
    return all.find(t => t.id === id) || null;
  }
  
  /**
   * 根据分类获取模板
   */
  getTemplatesByCategory(category: string): TableTemplate[] {
    const all = this.getAllTemplates();
    return all.filter(t => t.category === category);
  }
  
  /**
   * 搜索模板
   */
  searchTemplates(keyword: string): TableTemplate[] {
    const all = this.getAllTemplates();
    const lowerKeyword = keyword.toLowerCase();
    
    return all.filter(t => 
      t.name.toLowerCase().includes(lowerKeyword) ||
      t.description.toLowerCase().includes(lowerKeyword) ||
      t.tags.some(tag => tag.toLowerCase().includes(lowerKeyword))
    );
  }
  
  /**
   * 应用模板
   */
  applyTemplate(
    template: TableTemplate,
    context: { tableName: string; dialect?: string }
  ): TableDefinition {
    // 增加使用次数（深拷贝避免修改原始对象）
    const tplCopy = JSON.parse(JSON.stringify(template));
    tplCopy.usageCount++;
    this.saveTemplateUsage(tplCopy);
    
    // 转换模板为表定义
    const table: TableDefinition = {
      name: context.tableName,
      columns: tplCopy.columns.map((col: any) => ({
        name: col.name,
        type: col.type,
        length: col.length,
        numScale: '',
        notNull: col.notNull,
        pri: col.isPrimaryKey || false,
        unique: col.unique || false,
        auto_increment: col.autoIncrement || false,
        value: col.defaultValue || '',
        remark: col.comment || '',
      })),
      indexes: template.indexes?.map(idx => ({
        name: idx.name,
        columns: idx.columns.map(c => ({ name: c.name, length: c.length })),
        unique: idx.unique,
        type: idx.type,
        comment: idx.comment,
      })) || [],
      constraints: [],
    };
    
    return table;
  }
  
  /**
   * 保存为模板
   */
  saveAsTemplate(
    table: TableDefinition,
    metadata: { name: string; description: string; category?: string; tags?: string[] }
  ): TableTemplate {
    const template: TableTemplate = {
      id: `custom_${Date.now()}_${Math.random().toString(36).slice(2, 11)}`,
      name: metadata.name,
      description: metadata.description,
      category: (metadata.category || 'custom') as TableTemplate['category'],
      tags: metadata.tags || [],
      usageCount: 0,
      createdAt: Date.now(),
      columns: table.columns.map(col => ({
        name: col.name,
        type: col.type,
        length: col.length?.toString(),
        notNull: col.notNull,
        isPrimaryKey: col.pri,
        autoIncrement: col.auto_increment,
        unique: col.unique,
        defaultValue: col.value,
        comment: col.remark || '',
      })),
      indexes: table.indexes.map(idx => ({
        name: idx.name,
        columns: idx.columns.map(c => ({ name: c.name, length: c.length })),
        unique: idx.unique,
        type: idx.type,
        comment: idx.comment,
      })),
    };
    
    // 保存到 localStorage
    const userTemplates = this.getUserTemplates();
    userTemplates.push(template);
    localStorage.setItem('user_table_templates', JSON.stringify(userTemplates));
    
    return template;
  }
  
  /**
   * 更新用户模板
   */
  updateUserTemplate(id: string, updates: Partial<TableTemplate>): boolean {
    const userTemplates = this.getUserTemplates();
    const index = userTemplates.findIndex(t => t.id === id);
    
    if (index === -1) return false;
    
    userTemplates[index] = { ...userTemplates[index], ...updates };
    localStorage.setItem('user_table_templates', JSON.stringify(userTemplates));
    
    return true;
  }
  
  /**
   * 删除用户模板
   */
  deleteUserTemplate(id: string): boolean {
    const userTemplates = this.getUserTemplates();
    const filtered = userTemplates.filter(t => t.id !== id);
    
    if (filtered.length === userTemplates.length) {
      return false;
    }
    
    localStorage.setItem('user_table_templates', JSON.stringify(filtered));
    return true;
  }
  
  /**
   * 保存模板使用情况
   */
  private saveTemplateUsage(template: TableTemplate) {
    const builtinIndex = builtinTemplateData.findIndex(t => t.id === template.id);
    if (builtinIndex !== -1) {
      builtinTemplateData[builtinIndex].usageCount = template.usageCount;
    } else {
      this.updateUserTemplate(template.id, { usageCount: template.usageCount });
    }
  }
  
  /**
   * 导出模板
   */
  exportTemplate(id: string): string {
    const template = this.getTemplateById(id);
    if (!template) {
      throw new Error(`模板 ${id} 不存在`);
    }
    
    return JSON.stringify(template, null, 2);
  }
  
  /**
   * 导入模板
   */
  importTemplate(json: string): TableTemplate {
    try {
      const template = JSON.parse(json) as TableTemplate;
      
      // 生成新的 ID
      template.id = `custom_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      template.usageCount = 0;
      template.createdAt = Date.now();
      
      // 保存到用户模板
      const userTemplates = this.getUserTemplates();
      userTemplates.push(template);
      localStorage.setItem('user_table_templates', JSON.stringify(userTemplates));
      
      return template;
    } catch (error) {
      throw new Error('导入模板失败：JSON 格式错误');
    }
  }
  
  /**
   * 获取所有分类
   */
  getCategories(): string[] {
    const all = this.getAllTemplates();
    const categories = new Set(all.map(t => t.category));
    return Array.from(categories);
  }
  
  /**
   * 获取所有标签
   */
  getAllTags(): string[] {
    const all = this.getAllTemplates();
    const tags = new Set<string>();
    all.forEach(t => t.tags.forEach(tag => tags.add(tag)));
    return Array.from(tags);
  }
}

// 导出单例
export const templateService = new TemplateService();
