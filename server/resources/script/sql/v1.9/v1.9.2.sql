UPDATE `t_tag_tree` SET type = 5 WHERE type = 21 or type = 11;
UPDATE `t_tag_tree` SET code_path = REPLACE(code_path, '/11|', '/5|') WHERE type = 5;
UPDATE `t_tag_tree` SET code_path = REPLACE(code_path, '/21|', '/5|') WHERE type = 5 or type = 22;