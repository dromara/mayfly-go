UPDATE `t_sys_config`
SET
  `params` = '[{"name":"是否启用","model":"isUse","placeholder":"是否启用水印","options":"true,false"},{"name":"自定义信息","model":"content","placeholder":"额外添加的水印内容，可添加公司名称等"}]',
  `value` = '',
  `remark` = '水印信息配置',
  `key` = 'UseWatermark'
WHERE
  `key` = 'UseWartermark';