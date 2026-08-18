_alpha = 0;
level = 1;
time = 0;
switch(asdf)
{
   case 1:
      displaytext.text = _root.player1.hand1.gun.Name;
      break;
   case 2:
      displaytext.text = _root.player2.hand1.gun.Name;
      break;
   case 3:
      displaytext.text = _root.player3.hand1.gun.Name;
      break;
   case 4:
      displaytext.text = _root.player4.hand1.gun.Name;
}
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      _Y = _Y - 0.5;
      if(level == 1)
      {
         if(_alpha < 80)
         {
            _alpha = _alpha + 10;
         }
         time += 1;
         if(time >= 40)
         {
            level = 2;
         }
      }
      if(level == 2)
      {
         _alpha = _alpha - 10;
      }
      if(_alpha <= 1 || _root.deleteeverything)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
