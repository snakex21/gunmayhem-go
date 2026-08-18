this.onRelease = function()
{
   number = 0;
   number += Math.floor(_xmouse / 25);
   number *= 5;
   number += Math.floor(_ymouse / 25) + 1;
   _parent.colornumber = number;
   _parent.update();
};
mouseover = false;
this.onEnterFrame = function()
{
   if(mouseover)
   {
      if(masking._alpha > 0)
      {
         masking._alpha -= 20;
      }
   }
   else if(masking._alpha < 100)
   {
      masking._alpha += 10;
   }
};
this.onRollOver = function()
{
   mouseover = true;
};
this.onRollOut = function()
{
   mouseover = false;
};
