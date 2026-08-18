timer = 0;
disabled = false;
_rotation = _root.gototestnumber;
randgun = Math.round(this._rotation);
_rotation = 0;
originx = _X;
originy = _Y;
if(_root.gototestnumber <= 6)
{
   this.swapDepths(1);
   removeMovieClip(this);
   delete this.onEnterFrame;
}
this.onEnterFrame = function()
{
   if(_root.gamemode == 2 || _root.gamemode == 4)
   {
      _X = -300;
      _Y = 0;
      _alpha = 0;
      this.swapDepths(1);
      delete this.onEnterFrame;
   }
   if(!disabled)
   {
      if(_alpha < 100)
      {
         _alpha = _alpha + 5;
      }
      if(this.hitTest(_root.activeplayers[0].frame))
      {
         _root.activeplayers[0].getgun(randgun);
         j = 0;
         while(j < 10)
         {
            _root.CP("fx_pickup",_X + _parent._x,_Y + _parent._y + 20,0);
            j++;
         }
         disabled = true;
         _X = -300;
         _Y = 0;
         _alpha = 0;
      }
   }
   else
   {
      timer += 1;
      if(timer >= 100)
      {
         gotoAndPlay(random(40) + 10);
         timer = 0;
         disabled = false;
         _X = originx;
         _Y = originy;
      }
   }
};
