btn_back.onRelease = function()
{
   if(!_root.fadeaway)
   {
      btn_back.useHandCursor = false;
      newmc = _root.attachMovie("fadeaway","fadeaway",_root.fadedepth);
      newmc.targetframe = 9;
      _root.gotomenu = false;
      _root.playsound("menu.wav");
   }
};
win1.maxx = 0;
win2.maxx = 0;
win3.maxx = 0;
win4.maxx = 0;
win1._x = -200;
win2._x = -200;
win3._x = -200;
win4._x = -200;
