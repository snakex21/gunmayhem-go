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
btn_credits.onPress = function()
{
   getURL("http://www.thekevingu.com",_blank);
};
btn_music.onPress = function()
{
   getURL("http://www.incompetech.com",_blank);
};
