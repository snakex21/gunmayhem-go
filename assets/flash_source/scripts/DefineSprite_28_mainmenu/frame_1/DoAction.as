this.swapDepths(_root.mainmenudepth);
targety = -50;
slide._alpha = 0;
this.onEnterFrame = function()
{
   slide._y += (targety - slide._y) / 2;
   if(_xmouse < btn1._x || _xmouse > btn1._x + btn1._width || _ymouse < btn1._y)
   {
      if(slide._alpha > 0)
      {
         slide._alpha -= 20;
      }
   }
   else if(slide._alpha < 80)
   {
      slide._alpha += 20;
   }
};
btn1.onRelease = function()
{
   if(!_root.fadeaway)
   {
      newmc = _root.attachMovie("fadeaway","fadeaway",_root.fadedepth);
      newmc.targetframe = 4;
      _root.slideprevx = 0;
      _root.playsound("menu.wav");
   }
};
btn2.onRelease = function()
{
   if(!_root.fadeaway)
   {
      newmc = _root.attachMovie("fadeaway","fadeaway",_root.fadedepth);
      newmc.targetframe = 9;
      _root.playsound("menu.wav");
   }
};
btn3.onRelease = function()
{
   if(!_root.fadeaway)
   {
      newmc = _root.attachMovie("fadeaway","fadeaway",_root.fadedepth);
      newmc.targetframe = 8;
      _root.playsound("menu.wav");
   }
};
btn4.onRelease = function()
{
   if(!_root.fadeaway)
   {
      newmc = _root.attachMovie("fadeaway","fadeaway",_root.fadedepth);
      newmc.targetframe = 5;
      _root.playsound("menu.wav");
   }
};
btn5.onRelease = function()
{
   if(!_root.fadeaway)
   {
      newmc = _root.attachMovie("fadeaway","fadeaway",_root.fadedepth);
      newmc.targetframe = 7;
      _root.playsound("menu.wav");
   }
};
btn6.onRelease = function()
{
   if(!_root.fadeaway)
   {
      getURL("http://armorgames.com",_blank);
   }
};
