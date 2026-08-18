stop();
switch(_name)
{
   case "p1":
      number = 0;
      break;
   case "p2":
      number = 1;
      break;
   case "p3":
      number = 2;
      break;
   case "p4":
      number = 3;
}
if(_root.pgsdata[number][1] == -1)
{
   gotoAndStop(2);
}
else
{
   text0.text = _root.pgsdata[number][0];
   text1.text = _root.pgsdata[number][1];
   if(parseInt(text1.text) > _parent.win1.maxx)
   {
      _parent.win1.maxx = parseInt(text1.text);
      _parent.win1._x = _X;
   }
   else if(parseInt(text1.text) == _parent.win1.maxx)
   {
      _parent.win1.maxx = parseInt(text1.text);
      _parent.win1._x = -200;
   }
   text2.text = _root.pgsdata[number][2];
   text3.text = Math.round(parseInt(text1.text) / parseInt(text2.text) * 100) / 100;
   if(parseInt(text1.text) / parseInt(text2.text) > 100000)
   {
      text3.text = text1.text;
   }
   if(isNaN(parseInt(text1.text) / parseInt(text2.text)))
   {
      text3.text = 0;
   }
   if(parseInt(text3.text) > _parent.win2.maxx)
   {
      _parent.win2.maxx = parseInt(text3.text);
      _parent.win2._x = _X;
   }
   else if(parseInt(text3.text) == _parent.win2.maxx)
   {
      _parent.win2.maxx = parseInt(text3.text);
      _parent.win2._x = -200;
   }
   text4.text = _root.pgsdata[number][3];
   text5.text = _root.pgsdata[number][4];
   percentage = Math.round(parseInt(text5.text) / parseInt(text4.text) * 100);
   text6.text = percentage + "%";
   if(percentage > _parent.win3.maxx)
   {
      _parent.win3.maxx = percentage;
      _parent.win3._x = _X;
   }
   else if(percentage == _parent.win3.maxx)
   {
      _parent.win3.maxx = percentage;
      _parent.win3._x = -200;
   }
   text7.text = _root.pgsdata[number][5];
   text8.text = _root.pgsdata[number][6];
   text9.text = _root.pgsdata[number][7] * percentage / 100;
   if(parseInt(text9.text) > _parent.win4.maxx)
   {
      _parent.win4.maxx = parseInt(text9.text);
      _parent.win4._x = _X;
   }
   else if(parseInt(text9.text) == _parent.win4.maxx)
   {
      _parent.win4.maxx = parseInt(text9.text);
      _parent.win4._x = -200;
   }
   if(isNaN(parseInt(text5.text) / parseInt(text4.text)))
   {
      text6.text = "0%";
      text9.text = 0;
   }
}
