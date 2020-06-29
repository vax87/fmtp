

---
create or replace package utl_num
is

function numFromStr(p_var in varchar2) 
return number;
	
function dec2oct (p_num in number) 
return varchar2;

end utl_num;
/
commit;
show errors;

create or replace package body utl_num 
is

function numFromStr(p_var in varchar2)
return number
is
   l_number number         := 0;
   l_rig    varchar2(10)   := '';
   l_format varchar2(100)  := '999999999999d999999999999';
   l_res    varchar2(100)  := '';
begin

   select   VALUE 
   into     l_rig 
   from     NLS_DATABASE_PARAMETERS 
   where    PARAMETER = 'NLS_NUMERIC_CHARACTERS';

   if substr(l_rig,1,1) = '.' 
   then
      l_res := replace(p_var,',','.');
   else
      l_res := replace(p_var,'.',',');
   end if;

   l_number := to_number(l_res,l_format,'NLS_NUMERIC_CHARACTERS='''||l_rig||'''');

   return l_number;

exception
   when others then return -1;
end numFromStr;

function dec2oct (p_num in number) 
return varchar2 
is
   l_octval varchar2(128);
   l_n2     number := p_num;
begin

   while (l_n2 > 0 )
   loop
      l_octval := mod(l_n2, 8) || l_octval;
      l_n2     := trunc(l_n2 / 8 );
   end loop;

   return lpad(nvl(l_octval,'0'),4,'0');
end dec2oct;

end utl_num;
/
commit;
show errors;
exit;
