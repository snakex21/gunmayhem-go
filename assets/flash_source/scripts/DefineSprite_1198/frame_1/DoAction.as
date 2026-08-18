btn_back.onRelease = function()
{
   if(!_root.fadeaway)
   {
      btn_back.useHandCursor = false;
      newmc = _root.attachMovie("fadeaway","fadeaway",_root.fadedepth);
      newmc.targetframe = 10;
      _root.gotomenu = true;
      _root.playsound("menu.wav");
   }
};
