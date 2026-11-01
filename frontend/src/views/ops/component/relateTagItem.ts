import { Rules } from '@/common/rule';
import type { AutoFormData, AutoFormItemOf } from '@/components/auto-form/types';

/**
 * 资源分组（关联标签 tagCodePaths）字段声明 —— 所有资源编辑表单的唯一出处。
 *
 * 资源分组决定资源归属与团队隔离，是资源自身的必填属性（后端各资源入参的 tagCodePaths
 * 一律 binding:"required"），故前端同源声明必填：标签旁星号 + 提交前校验空数组。
 *
 * 收口在工厂里而不是各表单逐个书写，是因为漏声明的表现很隐蔽：分组留空时前端照常放行，
 * 直到提交被后端 400 拒绝才暴露（且提示不是字段级中文文案）。新增资源类型时直接引用本工厂即可。
 *
 * 控件由各表单提供同名插槽（<template #tagCodePaths="{ form }">）渲染 TagTreeSelect；
 * 字段走插槽渲染，故必填规则只能经 rules 声明（type='custom' 的默认文案是「请输入」，
 * 而这里是下拉式多选树，取 Rules.requiredSelect 得到「请选择资源分组」）。
 */
export const relateTagItem = <TForm extends AutoFormData = AutoFormData>(): AutoFormItemOf<TForm> => ({
    prop: 'tagCodePaths',
    label: 'tag.relateTag',
    type: 'custom',
    rules: [Rules.requiredSelect('tag.relateTag')],
});
