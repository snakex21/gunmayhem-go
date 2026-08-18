stop();
info.gotoAndStop(number);
btn_start.onRelease = function()
{
   if(!_root.fadeaway)
   {
      btn_start.useHandCursor = false;
      _parent.startmission();
   }
   _root.playsound("menu.wav");
};
